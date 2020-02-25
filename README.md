# snapcraft-warn-plugs-part

# WIP

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

apps:
  hello:
    command: bin/hello
    command-chain: [bin/check]
    adapter: full
  universe:
    command: bin/hello -g "Hello, universe!"
    adapter: legacy

parts:
  gnu-hello:
    source: http://ftp.gnu.org/gnu/$SNAPCRAFT_PROJECT_NAME/$SNAPCRAFT_PROJECT_NAME-$SNAPCRAFT_PROJECT_VERSION.tar.gz
    plugin: autotools

  check-plugs:
    source: https://github.com/ryanjyoder/snapcraft-warn-plugs-part.git
    source-type: git
    plugin: go


```