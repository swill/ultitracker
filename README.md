UltiTracker
===========

A simple web application to help teams track their players workouts and training habits.

It features a whitelabel system which allows for the presentation and configuration to be defined at build time.  Deployments are a single binary and I have included a `supervisord.conf` to configure a `program` to run the binary.  The included scripts will run the binary, on a server, from the HOME directory as the configured user.


BUILD DEPENDANCIES
------------------

In order to build the theming for the whitelabel feature of this application you will need `less` and `less-plugin-clean-css`.

**Install `less`**
``` bash
npm install -g less
```

**Install `less-plugin-clean-css`**
``` bash
npm install -g less-plugin-clean-css
```


RUN DEPENDANCIES
----------------

UltiTracker uses Google Sheets as its database.  Both the application configuration and the data entered into the application are stored in Google Sheets.

In order to run this application, you need to first configure a Google Sheet, create a Google API project, generate API credentials and then give access to the API user to the previously created Sheet.

`... details incoming ...`


DEPLOY DEPENDENCIES
-------------------

On the instance you plan to run the service on, assuming you are using `supervisord`, you will need to do the following.

```bash
sudo apt install supervisor
```

Update the `/etc/supervisor/supervisord.conf` file to include the following, where `cca-user` is the name of the user and group on the machine.
```bash
[unix_http_server]
file=/var/run/supervisor.sock   ; (the path to the socket file)
chmod=0766                       ; sockef file mode (default 0700)
chown=cca-user:cca-user
```

Reload supervisor with the new config.
```bash
/etc/supervisor/supervisord.conf
```

RUN / BUILD / DEPLOY
--------------------

The majority of the details to run, build and deploy UltiTracker have been scripted with bash for your convenience.

Be sure your `config` directory is setup, you have configured `__config.sh` and you have your credentials downloaded as `google-service-account.json` (configurable).

**_run.sh <team>**

`_run.sh` is mainly used as your development environment.  It will take the configured team and build the binary and run it locally.  This is how you should be developing a new team theme.  The theme is built into the binary to simplify deployment.

**_build.sh <team>**

`_build.sh` does basically the same thing as `_run.sh`, but it cross compiles the binaries into the `bin` directory.

**_deploy.sh <team>**

`_deploy.sh` transfers the themed binary, the application configuration and a supervisord configuration file to a server of your choice.  The included deploy configuration was developed for a Ubuntu VM.


