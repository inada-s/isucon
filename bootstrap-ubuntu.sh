# INSATALL tools
sudo apt-get update
sudo apt-get install -f -y git vim curl wget tmux git tree make python-dev python-pip python3-dev python3-pip unzip zip graphviz 

# INSTALL FlameGraph
wget https://github.com/brendangregg/FlameGraph/archive/master.zip
unzip master.zip
sudo mv FlameGraph-master/* /usr/local/bin/
rm -rf FlameGraph-master

# INSTALL golang
wget https://storage.googleapis.com/golang/go1.7.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.7.linux-amd64.tar.gz
rm go1.7.linux-amd64.tar.gz

GOPATH=~/go
GOBIN=~/go/bin
mkdir -p ~/go
cat << EOF >> ~/.profile
export PATH=$PATH:/usr/local/go/bin:$GOBIN
export GOPATH=$GOPATH
EOF

# SETUP vim
curl https://raw.githubusercontent.com/Shougo/neobundle.vim/master/bin/install.sh > neobundle-install.sh
sh ./neobundle-install.sh
rm neobundle-install.sh

BUNDLE_DIR=~/.vim/bundle
cat << EOF2 > ~/.vimrc
"------------------------------------------------------------------------------
" NeoBundle Scripts
"------------------------------------------------------------------------------
if has('vim_starting')
  if &compatible
    set nocompatible               " Be iMproved
  endif

  " Required:
  set runtimepath+=$BUNDLE_DIR/neobundle.vim/"
endif

" Required:
call neobundle#begin(expand('$BUNDLE_DIR'))"

" Let NeoBundle manage NeoBundle
" Required:
NeoBundleFetch 'Shougo/neobundle.vim'

" Add or remove your Bundles here:
NeoBundle 'fatih/vim-go'
NeoBundle 'vimplugin/project.vim'
NeoBundle 'Shougo/neocomplete'
NeoBundle 'ctrlpvim/ctrlp.vim'

" You can specify revision/branch/tag.
"NeoBundle 'Shougo/vimshell', { 'rev' : '3787e5' }

" Required:
call neobundle#end()

" Required:
filetype plugin indent on

" If there are uninstalled bundles found on startup,
" this will conveniently prompt you to install them.
NeoBundleCheck

"------------------------------------------------------------------------------
" General
"------------------------------------------------------------------------------

"
exe "set rtp+=".globpath(\$GOPATH, "src/github.com/nsf/gocode/vim")
set completeopt=menu,preview
set number
syntax on
set bs=start,indent
set encoding=utf8
set expandtab
set ffs=unix,dos,mac
let mapleader = "\<Space>"
let g:mapleader = "\<Space>"

"vim -b Option binary mode setting
augroup BinaryXXD
        autocmd!
        autocmd BufReadPre  *.bin let &binary =1
        autocmd BufReadPost * if &binary | silent %!xxd -g 1
        autocmd BufReadPost * set ft=xxd | endif
        autocmd BufWritePre * if &binary | %!xxd -r | endif
        autocmd BufWritePost * if &binary | silent %!xxd -g 1
        autocmd BufWritePost * set nomod | endif
augroup END

"Auto quickfix
au QuickfixCmdPost make,grep,grepadd,vimgrep copen
"Fix Quickfix color
hi Search cterm=NONE ctermfg=grey ctermbg=blue

"------------------------------------------------------------------------------
" NeoComplete
"------------------------------------------------------------------------------

" Disable AutoComplPop.
let g:acp_enableAtStartup = 0

" Use neocomplete.
let g:neocomplete#enable_at_startup = 1

" Use smartcase.
let g:neocomplete#enable_smart_case = 1

" Set minimum syntax keyword length.
let g:neocomplete#sources#syntax#min_keyword_length = 3
let g:neocomplete#lock_buffer_name_pattern = '\*ku\*'

" Close popup by <Space>.
" inoremap <expr><Space> pumvisible() ? neocomplete#close_popup() : "\<Space>"

" Plugin key-mappings.
inoremap <expr><C-g>     neocomplete#undo_completion()
"inoremap <expr><C-l>     neocomplete#complete_common_string()

" Recommended key-mappings.
" <CR>: close popup and save indent.
inoremap <silent> <CR> <C-r>=<SID>my_cr_function()<CR>
function! s:my_cr_function()
  return pumvisible() ? neocomplete#close_popup() : "\<CR>"
endfunction
" <TAB>: completion.
inoremap <expr><TAB>  pumvisible() ? "\<C-n>" : "\<TAB>"
" <C-h>, <BS>: close popup and delete backword char.
inoremap <expr><C-h> neocomplete#smart_close_popup()."\<C-h>"
inoremap <expr><BS> neocomplete#smart_close_popup()."\<C-h>"
inoremap <expr><C-y>  neocomplete#close_popup()
inoremap <expr><C-e>  neocomplete#cancel_popup()
" Close popup by <Space>.
"inoremap <expr><Space> pumvisible() ? neocomplete#close_popup() : "\<Space>"

" AutoComplPop like behavior.
let g:neocomplete#enable_auto_select = 1

" Enable omni completion.
autocmd FileType css setlocal omnifunc=csscomplete#CompleteCSS
autocmd FileType html,markdown setlocal omnifunc=htmlcomplete#CompleteTags
autocmd FileType javascript setlocal omnifunc=javascriptcomplete#CompleteJS
autocmd FileType python setlocal omnifunc=pythoncomplete#Complete
autocmd FileType xml setlocal omnifunc=xmlcomplete#CompleteTags

" Enable heavy omni completion.
"if !exists('g:neocomplete#sources#omni#input_patterns')
"  let g:neocomplete#sources#omni#input_patterns = {}
"endif
"let g:neocomplete#force_omni_input_patterns.go = '[^.[:digit:] *\t]\.'
if !exists('g:neocomplete#force_omni_input_patterns')
  let g:neocomplete#force_omni_input_patterns = {}
endif
let g:neocomplete#force_omni_input_patterns.go = '[^.[:digit:] *\t]\.'

"let g:neocomplete#sources#omni#input_patterns.php = '[^. \t]->\h\w*\|\h\w*::'
"let g:neocomplete#sources#omni#input_patterns.c = '[^.[:digit:] *\t]\%(\.\|->\)'
"let g:neocomplete#sources#omni#input_patterns.cpp = '[^.[:digit:] *\t]\%(\.\|->\)\|\h\w*::'

"------------------------------------------------------------------------------
" Vim-go
"------------------------------------------------------------------------------
let g:go_fmt_fail_silently = 1
let g:go_fmt_command = "goimports"
"let g:go_fmt_command = "gofmt"
let g:go_fmt_autosave = 1 
let g:go_highlight_functions = 1
let g:go_highlight_methods = 1
let g:go_highlight_fields = 1
let g:go_highlight_types = 1
let g:go_highlight_operators = 1
let g:go_highlight_build_constraints = 1

au FileType go nmap <leader>f :GoFmt<CR>
au FileType go nmap <leader>d :GoDef<CR>
au FileType go nmap <leader>e :e #<CR>
au FileType go nmap <leader>b :GoBuild<CR>
au FileType go nmap <leader>i :GoImports<CR>
EOF2

source ~/.profile
vim +NeoBundleInstall  +qall
vim +GoInstallBinaries +qall
