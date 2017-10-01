fl [![License](http://img.shields.io/:license-gpl3-blue.svg)](http://www.gnu.org/licenses/gpl-3.0.html) [![Build Status](https://travis-ci.org/opennota/fl.png?branch=master)](https://travis-ci.org/opennota/fl)
==

fl is a reverse proxy to the Flibusta e-library via Tor or I2P.

## Install

    go get -u github.com/opennota/fl

## Use

When invoked without parameters, `fl` chooses which network to use. First it tries to connect to Tor on port 9050 on the local machine and, failing that, switches to I2P on port 4444 on the local machine. After that it starts listening for requests on port 8080 and proxies them through the selected anonymity network to Flibusta.

For the options run

    fl -help
