# Saseum

```
   /|       |\
`__\\       //__'
   ||      ||
 \__`\     |'__/
   `_\\   //_'
   _.,:---;,._
   \_:     :_/
     |@. .@|
     |     |
     ,\.-./ \
     ;;`-'   `---__________-----.-.
     ;;;                         \_\
     ';;;                         |
      ;    |                      ;
       \   \     \        |      /
        \_, \    /        \     |\
          |';|  |,,,,,,,,/ \    \ \_
          |  |  |           \   /   |
          \  \  |           |  / \  |
           | || |           | |   | |
           | || |           | |   | |
           | || |           | |   | |
           |_||_|           |_|   |_|
          /_//_/           /_/   /_/
```

### Prepare

#### postgres

```bash
# https://github.com/pgvector/pgvector
cd /tmp
git clone --branch v0.8.2 https://github.com/pgvector/pgvector.git
cd pgvector
# creates vector.so file
make
# adds vector.so to installed postgres
sudo make install
```

#### NOTE:

[Check GoMLX ReadMe](https://github.com/gomlx/gomlx#-faq)

```bash
# to use nvidia
export GOMLX_BACKEND="xla:cuda"
```

todo:
- fix embedder model inference go routine need to create its own.
- change order consistency of util func of map
