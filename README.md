# A Windows service with endpoint for uploading a new version of itself

Basically, this is a Windows service with an http interface for uploading a file, in this case, a new version/build of the service. The service will then save this file, then call MoveFileEx API to ask Windows to overwrite the running binary with the newly uploaded one after system reboot.

# How to use it

#### Build and install the service

```
cd service
go build
service.exe install
service.exe start
```

#### Query the running service's version

You can do this using a browser. Navigate to `http://ip-address:8080/api/v1/version`. It should print the current version.

#### Create a new build for the service

Update the internal version string in `sevice\main.go`. Then build it.

#### Use the client to upload the new service build

```
cd client
go build
client.exe --file "path-of-the-new-service-build" --url http://ip-address:8080/api/v1/update/self self
```

This should upload the new file, then the target system will reboot after upload.

#### Confirm updated service binary after reboot

You can do this using a browser. Navigate to `http://ip-address:8080/api/v1/version`. It should print the updated version.

# ETW Logging

Logging uses ETW. For more information, check out this [project](https://github.com/flowerinthenight/go-windows-service-etw).

# License

[The MIT License](./LICENSE.md)
