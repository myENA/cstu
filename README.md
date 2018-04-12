# Cloudstack Template Updater
A small project to upload cloudstack templates, we use it as apart of a template update process. Creation of this cli app 
was driven by the desire to automate cloudstack template builds and updates. 


## Requirements
- Docker 17.0.6 ~ 18.0

## What it does
The cstu cli app can accept a config file, generated with `cstu init`, or through flags `cstu upload --help`.

- Check the passed os name and try to find an osTypeID, fail if it can't
- Check the passed zone name and try to find a zoneID, fail if it can't
- Creates an httpd container running on port 80
    - this process also moves the passed template file to /opt/cows/
- Checks CloudStack to see if there is an existing template with the same name, if true saves the ID for deletion later
- Makes the registerTemplate request to CloudStack
- Waits for the new template to be ready for use
- Deletes the older template if one was found
- Cleans up the docker container
- Can also download templates as long as they are extractable `cstu dl --help`
- When downloading a template, cstu writes a template.yml file based on the metadata it gets back from CloudStack. It also grabs the host IP address

```bash
$ cstu upload --configFile template.yml
2018-04-09T13:27:02-05:00 |INFO| Checking Options
2018-04-09T13:27:02-05:00 |INFO| Config File: template.yml
2018-04-09T13:27:02-05:00 |INFO| Reading config file at template.yml
2018-04-09T13:27:02-05:00 |INFO| Getting os id for CentOS 7
2018-04-09T13:27:02-05:00 |INFO| Getting Zone id for QA-ZONE-02
2018-04-09T13:27:02-05:00 |INFO| Creating httpd container
2018-04-09T13:27:02-05:00 |INFO| Running web server for upload: http://10.103.0.125
2018-04-09T13:27:03-05:00 |INFO| Checking if template Docker2 exists
2018-04-09T13:27:03-05:00 |INFO| Found a template with the same Name, saving ID for deletion later
2018-04-09T13:27:03-05:00 |INFO| Registering template at url: http://10.103.0.125/Docker2.qcow2
2018-04-09T13:27:03-05:00 |INFO| Grabbing new template ID
2018-04-09T13:27:03-05:00 |INFO| Waiting for new template to be ready
2018-04-09T13:27:03-05:00 |INFO| Checking if template Docker2 is ready: false
2018-04-09T13:27:18-05:00 |INFO| Checking if template Docker2 is ready: false
2018-04-09T13:27:33-05:00 |INFO| Checking if template Docker2 is ready: false
2018-04-09T13:27:48-05:00 |INFO| Checking if template Docker2 is ready: false
2018-04-09T13:28:03-05:00 |INFO| Checking if template Docker2 is ready: false
2018-04-09T13:28:18-05:00 |INFO| Checking if template Docker2 is ready: false
2018-04-09T13:28:33-05:00 |INFO| Checking if template Docker2 is ready: false
2018-04-09T13:28:48-05:00 |INFO| Checking if template Docker2 is ready: true
2018-04-09T13:28:48-05:00 |INFO| Deleting old template id 0d6dbd79-ab9d-4636-97d8-8ff9b4bfbca4
2018-04-09T13:28:48-05:00 |INFO| Stopping the httpd container
2018-04-09T13:28:48-05:00 |INFO| Your new Template Docker2 with ID faa6300c-e8d4-46d7-be12-ef48aa77e728 is ready for use
```

## Installing

Grab the latest release from [HERE](https://github.com/myENA/cstu/releases/)

## Running

##### With config file
Generate an empty config file to use: 
```bash
cstu init
cstu upload --configFile template.yml
```
##### Without config file
```bash
cstu upload --zone ZONE_NAME --template /path/to/template --format qcow2 --hypervisor kvm \
  --url http://cs/api/url --api-key APIKEY --secret-key SECRETKEY \
  --name TemplateName --host-ip YourHostIP --os "CentOS 7" \
  --displayText "Centos Docker Image"
```
##### Download template
```bash
$ ./cstu dl --templateID f7575949-22f1-4818-ab5e-462f3f4476af --zoneID 34b1a1a0-a8ba-47d1-a074-fb29bd5fe5f2

2018-04-11T13:11:23-05:00 |INFO| Downloading template from http://10.172.0.4/userdata/bbe010ed-7301-481a-a3f0-19ba2dfbe45d.qcow2
2018-04-11T13:11:43-05:00 |INFO| Wrote template  size: 1588461568
```
## Build

```bash
go get -d github.com/myENA/cstu
cd $GOPATH/src/github.com/myENA/cstu
glide i --strip-vendor
go build
```