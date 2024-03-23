## nvim-watcher
[WIP]

LazyVim often causes ERR_NO_FILES errors on my M2 Mac Mini, now that my neovim configuration has reached over 260 plugins, including dependencies. An interesting side-effect of this was the need to switch to Kitty terminal, after years of using Wezterm. Kitty solved the speed issue, and it runs just as fast as the purposely feature-lacking Alacritty on my computer.

The 2 main problems are:
* Neovim almost _always_ crashes often when trying to update the plugin list.
* AI plugins have runaway and ghost processses that need to be supervised.

I chose go because I wanted to write something that could perform work in parallel easily.
