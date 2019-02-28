# littr
<img src="http://peterstirrup.co.uk/wp-content/uploads/2019/01/littr-1.png">
A simple Go tool that times programs.

## Build & Run
Make sure you're in /cmd/littr
```
$ go build && ./littr -file route_to_file
```

### Before you build...
- Make sure the variable ```goLocation``` is pointing to your Go installation (```whereis go```)
- The ```path_to_file``` is from root directory (i.e. not in cmd)

## Flags
```$ ./littr -help```

##To do
- Allow littr to time servers and ongoing processes
- Change save file to new file (instead of overwrite)