# Overview

A simple, cron-like Windows service with an http interface. I use this tool to manage our deployment servers, gitlab runners (all VM's are running Windows), test automation VM's, etc. A client tool [`n1`](https://github.com/flowerinthenight/n1) is also available to interface with this service.

# Main functions

## Schedule commands to run periodically

Prior to this service, I have been using the task scheduler for running periodic tasks. Over time, it proved to be cumbersome to manage especially with lots of VM's involved.

This service runs command lines periodically as its main function. A `run.conf` configuration file is provided. The lowest timer tick value support is 1 minute. Read the `run.conf` comments for more information.

## Update self

This is the main reason why I wrote this service; to rid of logging in to every VM and do stuff.

You can use `n1` to upload a new version of this service to itself.

```
n1.exe update --file [new-service-exe] --hosts [ip1, ip2, ip3, ...] self
```

With this command, `n1` will upload the file, service saves it, then call `MoveFileEx` API with the `MOVEFILE_DELAY_UNTIL_REBOOT` flag to ask Windows to overwrite the running binary with the newly uploaded one after system reboot. By default, the service will reboot the system after calling `MoveFileEx`.

## Update all gitlab runners

Another useful feature; updating all gitlab runner services. Currently, all our runners are installed in `c:\runner` folder so this path is hardcoded.

```
n1.exe update --file [new-gitlab-runner-exe] --hosts [ip1, ip2, ip3, ...] runner
```

## Update run.conf

Same with updating the service itself, without the reboot.

```
n1.exe update --file [new-gitlab-runner-exe] --hosts [ip1, ip2, ip3, ...] conf
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

# ETW logging

Logging uses ETW. For more information, check out this [project](https://github.com/flowerinthenight/go-windows-service-etw).

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

## Uninstall the manifest file

```
admin_prompt> wevtutil.exe um jytrace.man
```

# License

[The MIT License](./LICENSE.md)
