# snapcraft-warn-plugs-part

# WIP
This binary can be included in a snap to warn users about disconnected interfaces. You can included customized messages explaining why the plug is needed.

## Usage
At this repo as a part to your snapcraft.yaml, and then include it in your command-chain.

```
name: hello
base: core
confinement: strict
grade: stable

version: "2.10"
summary: GNU Hello
description: GNU hello prints a friendly greeting.
assumes: [command-chain]

apps:
  hello:
    command: bin/hello
    ### This will will enable the warnings on this app. ###
    command-chain: [bin/snapcraft-warn-plugs-part]
    plugs:
    - home
    - network
    - lxd

parts:
  gnu-hello:
    source: http://ftp.gnu.org/gnu/$SNAPCRAFT_PROJECT_NAME/$SNAPCRAFT_PROJECT_NAME-$SNAPCRAFT_PROJECT_VERSION.tar.gz
    plugin: autotools
  
  ### Build the part here ###
  check-plugs:
    source: https://github.com/ryanjyoder/snapcraft-warn-plugs-part
    source-type: git
    plugin: go
    go-importpath: github.com/ryanjyoder/snapcraft-warn-plugs-part
```

You can customize the warning message by including a `plugs.yaml` file in the root of the snap.
```
network:
  required: false
  reason: Required to download international translations 
lxd:
  required: true
  reason: Required to run greeting workloads

```

Any plugs that are marked as `required` will be shown every time the snap is run until it is connected. Warning messages for plugs that are not required will only appear once the first time the snap is run.