# libp2p-ftp
Simple ftp via libp2p-go

```
$ p2pftp --help
NAME:
   p2pftp - simple p2pftp application

USAGE:
   p2pftp [global options] command [command options] [arguments...]

VERSION:
   0.0.0

COMMANDS:
     listen   listen as ftp server
     connect  connect to remote peer
     list     list files under given directory
     put      put file name to remote directory
     get      get remote file
     delete   delete remote file
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --conf value, -c value  configure file name with whole path (default: "/etc/libp2p-ftp/conf.json")
   --help, -h              show help
   --version, -v           print the version
```
