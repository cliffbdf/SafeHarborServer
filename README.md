# SafeHarborServer
Server that provides REST API for the SafeHarbor system.
## Scan Providers
### Clair
### Twistlock
### OpenScap

## Design and REST API
See https://drive.google.com/open?id=1r6Xnfg-XwKvmF4YppEZBcxzLbuqXGAA2YCIiPb_9Wfo
## To Build Code
1. Go to the <code>build/Centos</code> directory.
2. Run <code>vagrant up</code>

## To Deploy
1. Go to the <code>deploy/</code>(target-OS) directory.
2. Run <code>make -f ../../certs.mk</code> (if you have not already done this)
3. Edit <code>safeharbor.conf</code> (usually does not need to change)
4. Run <code>./deploy.sh</code>
5. Log into the server using <code>vagrant ssh</code>.
6. Edit <code>conf.json</code> (usually does not need to change)
7. Edit <code>auth_config.yml</code> (usually does not need to change)
8. Log out of the server.

## To Start
<code>./start.sh</code>

## To Stop
<code>./stop.sh</code>
 trigger
