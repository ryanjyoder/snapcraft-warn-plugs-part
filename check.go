package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const (
	ALREADY_WARNED    = "already_warned_plugs"
	WARNINGS_DISABLED = "plug_warnings_disabled"
	PLUG_DESCRIPTIONS = "plugs.yaml"

	MSG_OPTIONAL_PLUGS = "Consider connecting the following interfaces:"
	MSG_REQUIRED_PLUGS = "The follow interfaces must be connected for app to work properly:"
)

type warningState struct {
	// If true the other values of the struct should be ignored
	WarningsDisabled bool

	// The user has already be warned about disconnected plugs.
	// Do not warn again about optional plugs
	AlreadyWarned bool

	// If AlreadyWarned is true then the optional plugs maybe be empty
	OptionalPlugs map[string]PlugStatus
	RequiredPlugs map[string]PlugStatus
}

type PlugStatus struct {
	// This interface is not auto-connect but the author has indicated that the app cannot work without this plug.
	Required bool `yaml:"required"`

	// A human readable explanation for why the snap needs this interface.
	Reason string `yaml:"reason"`

	Connected bool
}

func main() {
	snapDir := os.Getenv("SNAP")
	if snapDir == "" {
		fatal("Error: Could not read SNAP environment variable")
	}

	userDataDir := os.Getenv("SNAP_USER_DATA")
	if userDataDir == "" {
		fatal("Error: Could not read SNAP_USER_DATA environment variable")
	}

	state, err := gatherState(userDataDir, snapDir)
	if err != nil {

	}

	msg := getWarnMessage(state)
	fmt.Fprintf(os.Stderr, msg)

	setWarnFlags(userDataDir, state)

	if len(os.Args) < 2 {
		return
	}

	cmdStr := os.Args[1]
	cmd := exec.Command(cmdStr, os.Args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	if err == nil {
		return
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	}
	os.Exit(1)
}

func getWarnMessage(state warningState) string {
	if state.WarningsDisabled {
		return ""
	}

	msg := ""
	// We have not uet warned about optional plugs, create the message for optional plugs
	if !state.AlreadyWarned {
		// if it's already we can skip it
		for plugName, plug := range state.OptionalPlugs {
			if plug.Connected {
				continue
			}
			msg += fmt.Sprintf("\t%s\t\t- %s\n", plugName, plug.Reason)
		}
	}
	// add header if there are some optional plugs
	if msg != "" {
		msg = MSG_OPTIONAL_PLUGS + "\n" + msg
	}

	requiredMsg := ""
	for plugName, plug := range state.RequiredPlugs {
		// skip connected interfaces
		if plug.Connected {
			continue
		}
		requiredMsg += fmt.Sprintf("\t%s\t\t- %s\n", plugName, plug.Reason)
	}
	if requiredMsg != "" {
		msg = msg + "\n" + MSG_REQUIRED_PLUGS + "\n" + requiredMsg
	}
	if msg != "" {
		msg += "\n\n"
	}
	return msg

}

func setWarnFlags(userDateDir string, state warningState) error {
	// If we have warned the used then this is our first time doing so. Set the flag
	if !state.AlreadyWarned {
		file, err := os.Create(filepath.Join(userDateDir, ALREADY_WARNED))
		if err != nil {
			return err
		}
		file.Close()
	}

	// warnings are disabled no flags left to set.
	if state.WarningsDisabled {
		return nil
	}

	// Check if any required connections are still disconnect we cannot yet disable warnings
	for _, plug := range state.RequiredPlugs {
		if !plug.Connected {
			return nil
		}
	}

	// If we're here ALL required plugs are connectded, we can now disable warnings
	file, err := os.Create(filepath.Join(userDateDir, WARNINGS_DISABLED))
	if err != nil {
		return err
	}
	file.Close()

	return nil
}

func gatherState(userDataDir string, snapDir string) (warningState, error) {
	state := warningState{}

	// First check if warnings are disabled, so we can bail out early if they are.
	disabled, err := checkFileFlag(filepath.Join(userDataDir, WARNINGS_DISABLED))
	if err != nil {
		return state, err
	}
	state.WarningsDisabled = disabled

	// if warnings are disabled, bailout early.
	if state.WarningsDisabled {
		return state, nil
	}

	// Check if we've already warned the user about disconnected plugs
	warned, err := checkFileFlag(filepath.Join(userDataDir, ALREADY_WARNED))
	if err != nil {
		return state, err
	}
	state.AlreadyWarned = warned

	// check if the plugs.yaml is included in the snap
	plugsYamlExists, err := checkFileFlag(filepath.Join(snapDir, PLUG_DESCRIPTIONS))
	if err != nil {
		return state, err
	}

	// Load plug data
	requiredPlugs := map[string]PlugStatus{}
	optionalPlugs := map[string]PlugStatus{}
	// if plugs.yaml is included in snap load it first
	if plugsYamlExists {
		requiredPlugs, optionalPlugs, err = loadPlugsYaml(filepath.Join(snapDir, PLUG_DESCRIPTIONS))
		if err != nil {
			return state, err
		}
	}

	// if we've already warned the user clear any optional plugs
	if state.AlreadyWarned {
		optionalPlugs = map[string]PlugStatus{}
	}

	// if we have not warned the user also include any interfaces from the meta/snap.yaml
	if !state.AlreadyWarned {
		// also include any plugs not explicitly defined in the plugs.yaml
		err = loadFromSnapYaml(optionalPlugs, filepath.Join(snapDir, "meta", "snap.yaml"))
		if err != nil {
			return state, err
		}
	}

	// Finally check which plugs are connected.
	for plugName, plug := range requiredPlugs {
		isConnected, err := plugIsConnected(plugName)
		if err != nil {
			return state, err
		}
		plug.Connected = isConnected
		requiredPlugs[plugName] = plug
		delete(optionalPlugs, plugName) // make sure the plug doesn't show up twice
	}
	for plugName, plug := range optionalPlugs {
		isConnected, err := plugIsConnected(plugName)
		if err != nil {
			return state, err
		}
		plug.Connected = isConnected
		optionalPlugs[plugName] = plug
	}

	state.OptionalPlugs = optionalPlugs
	state.RequiredPlugs = requiredPlugs
	return state, nil
}

func checkFileFlag(filename string) (bool, error) {
	_, err := os.Stat(filename)
	// File exists, the flag is true
	if err == nil {
		return true, nil
	}

	// "File does not exist" errors are ok. The flag is false.
	if os.IsNotExist(err) {
		return false, nil
	}

	// All other errors are unexpected
	return false, err
}

func loadPlugsYaml(filename string) (requiredPlugs map[string]PlugStatus, optionalPlugs map[string]PlugStatus, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	yamlBytes, err := ioutil.ReadAll(file)

	plugs := map[string]PlugStatus{}
	requiredPlugs = map[string]PlugStatus{}
	optionalPlugs = map[string]PlugStatus{}
	err = yaml.Unmarshal(yamlBytes, &plugs)

	for plugName, plug := range plugs {
		if plug.Required {
			requiredPlugs[plugName] = plug
		} else {
			optionalPlugs[plugName] = plug
		}
	}

	return requiredPlugs, optionalPlugs, err
}

// snapYaml is used to unmarshal the SNAP/meta/snap.yaml
type snapYaml struct {
	Apps map[string]struct {
		Plugs []string `yaml:"plugs"`
	} `yaml:"apps"`
}

func loadFromSnapYaml(plugs map[string]PlugStatus, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	yamlBytes, err := ioutil.ReadAll(file)

	snap := snapYaml{}
	err = yaml.Unmarshal(yamlBytes, &snap)
	if err != nil {
		return err
	}

	for i := range snap.Apps {
		for _, plugName := range snap.Apps[i].Plugs {
			// don't override existing plug data
			if _, ok := plugs[plugName]; ok {
				continue
			}
			plugs[plugName] = PlugStatus{Required: false}
		}
	}

	return err
}

func plugIsConnected(plugName string) (bool, error) {
	cmd := exec.Command("snapctl", "is-connected", plugName)

	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if _, ok := err.(*exec.ExitError); ok {
		return false, nil
	}
	return false, err
}

func fatal(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
	os.Exit(1)
}
