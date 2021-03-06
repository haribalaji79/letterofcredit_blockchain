'use strict';
/*******************************************************************************
 * Copyright (c) 2015 IBM Corp.
 *
 * All rights reserved.
 *
 * This module assists with the user management for the blockchain network. It has
 * code for registering a new user on the network and logging in existing users.
 *
 * TODO refactor this into an object.
 *
 * Contributors:
 *   David Huffman - Initial implementation
 *   Dale Avery
 *
 * Created by davery on 3/16/2016.
 *******************************************************************************/

// Use a tag to make logs easier to find
const TAG = 'user_manager:';

/**
 * Whoever configures the hfc chain object needs to send it here in order for this user manager to function.
 * @param myChain The object representing our chain.
 */
module.exports = function (myChain, useTLS) {

    console.log(TAG, 'configuring user management');
    if (!myChain)
        throw new Error('User manager requires a chain object');
    let chain = myChain;
    let tls = useTLS;
    let manager = {};
    
    function getAllMethodNames(obj) {
    	  let methods = new Set();
    	  while (obj = Reflect.getPrototypeOf(obj)) {
    	    let keys = Reflect.ownKeys(obj)
    	    keys.forEach((k) => methods.add(k));
    	  }
    	  return methods;
    	}

    /**
     * Mimics a enrollUser process by attempting to register a given id and secret against
     * the first peer in the network. 'Successfully registered' and 'already logged in'
     * are considered successes.  Everything else is a failure.
     * @param enrollID The user to log in.
     * @param enrollSecret The secret that was given to this user when registered against the CA.
     * @param cb A callback of the form: function(err)
     */
    manager.login = function (enrollID, username, password, cb) {
        console.log(TAG, 'login() called');

        if (!chain) {
            cb(new Error('Cannot login a user before setup() is called.'));
            return;
        }

        console.log("Methods:", getAllMethodNames(chain));
        chain.login(enrollID, username, password, function (getError, usr) {
            if (getError) {
                console.log(TAG, 'login() failed for \"' + username + '\":', getError.message);
                if (cb) cb(getError);
            } else {
                console.log(TAG, 'Successfully logged in:', username);
                cb(getError, usr);
            }
        });
    };

    /**
     * Registers a new user in the membership service for the blockchain network.
     * @param enrollID The name of the user we want to register.
     * @param cb A callback of the form: function(error, user_credentials)
     */
    manager.registerUser = function (enrollID, cb) {
        console.log(TAG, 'registerUser() called');

        if (!chain) {
            cb(new Error('Cannot register a user before setup() is called.'));
            return;
        }

        chain.getMember(enrollID, function (err, usr) {
            if (!usr.isRegistered()) {
                console.log(TAG, 'Sending registration request for:', enrollID);
                // Hack to make registration work for local and bluemix blockchain networks
                let affiliation = 'institution_a';
                if (tls) affiliation = 'group1';
                let registrationRequest = {
                    enrollmentID: enrollID,
                    affiliation: affiliation
                };
                usr.register(registrationRequest, function (err, enrollSecret) {
                    if (err) {
                        cb(err);
                    } else {
                        let cred = {
                            id: enrollID,
                            secret: enrollSecret
                        };
                        console.log(TAG, 'Registration request completed successfully!');
                        cb(null, cred);
                    }
                });
            } else {
                cb(new Error('Cannot register an existing user'));
            }
        });
    };

    return manager;
};