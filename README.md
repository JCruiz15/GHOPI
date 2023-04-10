# GHOPI

#### Table of contents
 - [Prerequisites](#prerequisites)
 - [Installation guide](#installation-guide)
   - [Windows](#windows)
   - [Linux and MacOS](#linux-and-macos)
   - [Setting up .env file](#setting-up-env-file)
     - [Open Project app](#open-project-app)
     - [Github app](#github-app)
 - [Work in progress](#work-in-progress)
 - [License](#license)

---

Welcome to the GHOPI, our app aim is to create a channel of communication between Open Project and Github. This service must give project leaders and CEOs more freedom and let them focus on their work and avoid as much as possible unnecessary stops to change repository permissions, create new branches, open and close pull requests, etc. 

GHOPI makes use of the RESTful APIs of Open Project and Github to get information from them and synchronize both apps. This will be possible thanks to the webhooks provided by them which are easily configurable.

GHOPI is capable of creating branches in the repository for each task created on Open Project, give or remove permissions to people working on the tasks of each project, send a message into Open Project tasks when a Pull request is opened or closed, and many more features to reduce the waste of time.

It also has a web interface which will provide an easy configuration process and also logs viewer to check every information or error happening into the app.

![GHOPI's user interface, home page](./static/img/GHOPI_logo.svg)

## Prerequisites
[Go](https://go.dev/) version 1.19.1 or higher.
Any technology capable of make your app instance public.

## Installation guide

### Windows

To install this app in Windows just download the executable file `GHOPI.exe` and fill the .env file as explained below.

Then use your technology to launch the app publicly.

### Linux and MacOS

To install this app in Unix computers, clone this repository into your computer and execute the command:
 
```shell
go build main.go
```

Which will create an executable file to use the app. Then use your technology to launch the app publicly.

### Setting up .env file

To get this app to work you firstly need to set up a .env file which must include the client id and secret id of Github and Open Project to be able to log in them. This .env file must be in the same folder as the `GHOPI.exe` app or in the project root folder.

#### Open Project app

In your Open Project instance go to `Administration > Authentication > OAuth Applications`, where you can add a new application pushing the `Add` button.

In the configuration page opened just fill in the gaps with GHOPI's information, in the redirect URI be sure to write `https://your-app-url/op/login/callback` where `https://your-app-url` is your GHOPI's path which **must** be public.

![Open Project oauth set up](./static/img/OP_appsetup.png)

Once you have created it, the credentials will be shown and you will have to save the Client ID and the Client secret into the .env file with the names: `OPENPROJECT_CLIENTID` and `OPENPROJECT_SECRETID`.

![Open Project oauth set up credentials](./static/img/OP_appsetup_result.png)

#### Github app

To configure Github oauth go to `Settings > Developer settings > OAuth apps` and click the `new oauth app` button.

In the configuration page you must fill the Homepage URL field with `https://your-app-url` and the Authorization callback URL with `https://your-app-url/github/login/callback`, where `https://your-app-url` is your GHOPI's path which **must** be public.

![Github oauth set up](./static/img/GH_appsetup.png)

Once the app is created new client ID and Secret client will be created. You have to save them into the .env file with the names: `GITHUB_CLIENTID` and `GITHUB_SECRETID`.

![Github oauth set up credentials](./static/img/GH_appsetup_result.png)

## Work in progress

We are working on dockerizing the app so the installation is much easier. We are still documenting all the app code so it is more readable.

## License

This project is licensed under GNU v3.0 license. Read it in [LICENSE.md](https://github.com/JCruiz15/GHOPI/blob/main/LICENSE.md) file.
