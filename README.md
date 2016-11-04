# Overview

A simple, cron-like Windows service with an http interface. I use this tool to manage our deployment servers, gitlab runners (all VM's are running Windows), test automation VM's, etc. A client tool [`n1.exe`](https://github.com/flowerinthenight/n1) is also available to interface with this service.

# Main functions

## Schedule commands to run periodically

Prior to this service, I have been using the task scheduler for running periodic tasks. Over time, it proved to be cumbersome to manage especially with lots of VM's involved.

This service runs command lines periodically as its main function. A [`run.conf`](./run.conf) configuration file is provided. The lowest timer tick value support is 1 minute.

```
┌───────────── min (0 - 59)
│ ┌─────────── hour (0 - 23)
│ │ ┌───────── day of month (1 - 31)
│ │ │ ┌─────── month (1 - 12)
│ │ │ │ ┌───── day of week (0 - 6) (0 to 6 are Sunday to Saturday
│ │ │ │ │
* * * * * command to execute

Examples:

Run every minute:
* * * * * cmd.exe /arg1 /arg2 "arg with space" /sampledir "path\to\something"

Run every 5 minutes:
*/5 * * * * cmd.exe /arg1 /arg2 "arg with space" /sampledir "path\to\something"

Run every 2 hours:
* */2 * * * cmd.exe /arg1 /arg2 "arg with space" /sampledir "path\to\something"

Run every Aug 28 at 8:00am:
0 8 28 8 * file.exe -arg1 -arg2

Run every Saturday at 10 mins interval:
*/10 * * * 6 file.exe --arg1 --arg2
```

Check out [`run.conf`](./run.conf) configuration for more information. 

## Update self

This is the main reason why I wrote this service; to rid of logging in to every VM and do stuff.

You can use [`n1.exe`](https://github.com/flowerinthenight/n1) to upload a new version of this service to itself.

```
n1.exe update --file [new-service-exe] --hosts [ip1, ip2, ip3, ...] self
```

With this command, [`n1.exe`](https://github.com/flowerinthenight/n1) will upload the file, service saves it, then call [`MoveFileEx`](https://msdn.microsoft.com/en-us/library/windows/desktop/aa365240%28v=vs.85%29.aspx?f=255&MSPPError=-2147217396) API with the `MOVEFILE_DELAY_UNTIL_REBOOT` flag to ask Windows to overwrite the running binary with the newly uploaded one after system reboot. By default, the service will reboot the system after calling `MoveFileEx`.

## Update all gitlab runners

Another useful feature; updating all gitlab runner services. Currently, all our runners are installed in `c:\runner` folder so this path is hardcoded.

```
n1.exe update --file [new-gitlab-runner-exe] --hosts [ip1, ip2, ip3, ...] runner
```

## Update run.conf

Same with updating the service itself, without the reboot.

```
n1.exe update --file [new-run-conf-file] --hosts [ip1, ip2, ip3, ...] conf
```

## Upload file

I use this to upload additional tools/executables to add to `run.conf` but you can upload any file to any location using this command.

```
n1.exe upload --file [any-file] --path [location/path-to-copy-the-file] --host [ip]
```

## File stats

```
n1.exe stat --files [comma-separated files/dirs] --host [ip]
```

## Read file

I use this mainly to confirm whether the `run.conf` update process is successful or not.

```
n1.exe read --file [file-to-read] --host [ip]
```

## Execute commands remotely

Quite a dangerous feature, though. Remember that this service runs under SYSTEM account in session 0.

```
n1.exe exec --cmd [cmd-to-execute] --host [ip]
```

Since `cmd` will be executed from service session, it is not interactive by default. To run an interactive command, use [`n1.exe`](https://github.com/flowerinthenight/n1)'s `--interactive=true` option.

```
n1.exe exec --cmd [cmd-to-execute] --host [ip] --interactive=true --wait=true --waitms=5000
```

The service will run `cmd` within the same session as `winlogon.exe` (not session 0) via the [`CreateProcessAsUser`](https://msdn.microsoft.com/en-us/library/windows/desktop/ms682429%28v=vs.85%29.aspx?f=255&MSPPError=-2147217396) API. This is done through an external function [`StartSystemUserProcess`](https://github.com/flowerinthenight/win-cpplib/blob/master/libcore/libcore.cpp) hosted in [`libcore.dll`](https://github.com/flowerinthenight/win-cpplib).

## Query service version

I use this mainly to confirm whether the service update process is successful or not.

```
n1.exe version --host [ip]
```

# Installation

Run the following commands as administrator:

```
holly.exe install
holly.exe start
```

# Uninstall

Run the following commands as administrator:

```
holly.exe stop
holly.exe remove
```

# Troubleshooting

## Service cannot start

* `disptrace.dll` is built using VS2015 so you need to install VC++ 2015 redistributable.

## Cannot access http interface

* Add `holly.exe` file to 'Allowed apps' in your Windows Firewall. You can also enable port 8080 as well.

# ETW logging

Logging uses ETW. For more information, check out this [project](https://github.com/flowerinthenight/go-windows-service-etw) or this [blog series](http://flowerinthenight.com/blog/2016/03/01/etw-part1).

## Quickstart guide to view ETW logs (to aid in service debugging)

* Install manifest file

```
admin_prompt> wevtutil.exe im jytrace.man /mf:"full_path_to_disptrace.dll" /rf:"full_path_to_disptrace.dll"
```

* Use `mftrace.exe` to view real-time logs from service from command line/Powershell. You can find `mftrace.exe` from the Windows SDK/WDK installation folder (i.e. `C:\Program Files (x86)\Windows Kits\10\bin\x86\`). Note that `mftrace.exe` needs `mfdetours.dll` (from the same location) to run, in case you want to copy the two files to a separate location. For more information about `mftrace.exe`, check out this [link](https://msdn.microsoft.com/en-us/library/windows/desktop/ff685116(v=vs.85).aspx).

```
admin_prompt> mftrace.exe -c config.xml
```

If you're forking/contributing/modifying the service and you want more logs from the service, use the provided `trace()` wrapper function.

# License

[The MIT License](./LICENSE.md)
