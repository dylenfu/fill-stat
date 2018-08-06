#!/usr/bin/env bash

# vendor init
rm -rf $GOPATH/src/github.com/dylenfu/fill-stat/vendor
govendor init

# vendor add external libraries
govendor add +external

# copy go-ethrenum c libs
rm -rf $GOPATH/src/github.com/dylenfu/fill-stat/vendor/github.com/ethereum/go-ethereum/crypto/secp256k1
cp -r $GOPATH/src/github.com/Loopring/relay-cluster/vendor/github.com/ethereum/go-ethereum/crypto/secp256k1 $GOPATH/src/github.com/dylenfu/fill-stat/vendor/github.com/ethereum/go-ethereum/crypto/
