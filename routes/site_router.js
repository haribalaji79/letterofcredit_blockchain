'use strict';
/*******************************************************************************
 * Copyright (c) 2015 IBM Corp.
 *
 * All rights reserved.
 *
 * Handles the site routing and also handles the calls for user registration
 * and logging in.
 *
 * Contributors:
 *   David Huffman - Initial implementation
 *   Dale Avery
 *******************************************************************************/
var express = require('express');
var router = express.Router();
var setup = require('../setup.js');
var fs = require('fs');
// Load our modules.
var userManager;
var chaincode_ops;

// Use tags to make logs easier to find
var TAG = 'router:';

// ============================================================================================================================
// Home
// ============================================================================================================================
router.get('/', isAuthenticated, function (req, res) {
    console.log("User already authenticated");
});

router.get('/:page', function (req, res) {
	if (req.params.page == 'login') {
		res.render('login', {layout : null});
	} else if(req.params.page == 'getDocument'){
		console.log('Begin getDocument:' );
    	getDocument(req, res);
	} else if (req.params.page == 'logout') {
		req.session.destroy();
	    res.redirect('/login');
	} else if (req.params.page == 'lcList') {
		retrieveLCApplications(req, res);
	} else if (req.params.page == 'importerBank' || req.params.page == 'exporterBank'
			|| req.params.page == 'exporter' || req.params.page == 'customs') {
		if (req.query.shipmentId != null) {
			req.session.shipmentId = req.query.shipmentId;
			chaincode_ops.fetchLC('WebAppAdmin', req.query.shipmentId, function (err, lc) {
		        if (err) {
		            console.error(TAG, 'FetchLC failed:', err);
		            res.redirect('/lcList');
		        } else {
		        	var lcJSON = JSON.parse(lc.toString());  
		        	console.log("JSON : " + lc.toString());
		        	var documentNames = lcJSON.documentNames;
		        	var lcFileName, blFileName, insFileName;
		        	if (documentNames != null) {
		        		for (var i = 0; i < documentNames.length; i++) { 
			        		if (i == 0) {
			        			lcFileName = documentNames[0];
			        		} else if (i == 1) {
			        			blFileName = documentNames[1];
			        		} else if (i == 2) {
			        			insFileName = documentNames[2];
			        		}
			        	}
		        	}
		        	
		        	var ebState1 = (lcJSON.exporterBankApproved) ? 'inline' : 'none';
		        	var ebState2 = (!lcJSON.exporterBankApproved) ? 'inline' : 'none';
		        	var eDocState1 = (lcJSON.exporterDocsUploaded) ? 'inline' : 'none';
		        	var eDocState2 = (!lcJSON.exporterDocsUploaded) ? 'inline' : 'none';
		        	var cusState1 = (lcJSON.customsApproved) ? 'inline' : 'none';
		        	var cusState2 = (!lcJSON.customsApproved) ? 'inline' : 'none';
		        	var payState1 = (lcJSON.paymentComplete) ? 'inline' : 'none';
		        	var payState2 = (!lcJSON.paymentComplete) ? 'inline' : 'none'; 
		        	var state = {'ebState1' : ebState1, 'ebState2' : ebState2, 'eDocState1' : eDocState1, 'eDocState2' : eDocState2,
		        		'cusState1' : cusState1, 'cusState2' : cusState2, 'payState1': payState1, 'payState2' : payState2};
		        	var responsePage = req.params.page;
		        	var showCreate = (responsePage == 'importerBank') ? 'inline-block' : 'none';
		        	if (responsePage == 'importerBank') {
		        		if (lcJSON.customsApproved) {
		        			responsePage = 'importerBankPayment';
		        		}
		        	}
		        	var showCreate = 
		        	res.render(responsePage, {'lc' : lcJSON, disabled : 'disabled', 
		        		'lcFileName' : lcFileName, 'blFileName' : blFileName, 'insFileName' : insFileName,
		        		 role : req.session.role, date : formatDate(new Date()), st : state, createVisible : showCreate});
		        }
			})
		} else {
			res.render(req.params.page, {role : req.session.role, date : formatDate(new Date())});
		}
	}
	else {
		res.render(req.params.page, {role : req.session.role, date : formatDate(new Date())});
	}
});

function formatDate(date) {
	  var monthNames = [
	    "January", "February", "March",
	    "April", "May", "June", "July",
	    "August", "September", "October",
	    "November", "December"
	  ];
	  
	  var daysOfWeek = [
		"Sunday", "Monday", "Tuesday", "Wednesday",
		"Thursday", "Friday", "Saturday"               
	  ];
	
	  var day = date.getDate();
	  var monthIndex = date.getMonth();
	  var year = date.getFullYear();
	  var week = date.getDay();
	  console.log("Getting Date week : " + week);
	
	  return daysOfWeek[week] + ', ' + day + ' ' + monthNames[monthIndex] + ' ' + year;
}

// function to create file from base64 encoded string
function base64_decode(base64str) {
    // create buffer object from base64 encoded string, it is important to tell the constructor that the string is base64 encoded
    var bitmap = new Buffer(base64str, 'base64');
    return bitmap;
}

// convert image to base64 encoded string
/*var base64str = base64_encode('kitten.jpg');
console.log(base64str);*/

router.post('/processLogin', function (req, res) {
    login(req, res);
});

module.exports = router;

module.exports.setup_helpers = function(configured_chaincode_ops, user_manager) { 
    if(!configured_chaincode_ops)
        throw new Error('Router needs a chaincode helper in order to function');
    chaincode_ops = configured_chaincode_ops;
    userManager = user_manager;
};

function isAuthenticated(req, res, next) {
    if (!req.session.username || req.session.username === '') {
        console.log(TAG, '! not logged in, redirecting to login');
        return res.redirect('/login');
    }

    console.log(TAG, 'user is logged in');
    next();
}

/**
 * Handles form posts for registering new users.
 * @param req The request containing the registration form data.
 * @param res The response.
 */
function register(req, res) {
    console.log('site_router.js register() - fired');
    req.session.reg_error_msg = 'Registration failed';
    req.session.error_msg = null;

    // Determine the user's role from the username, for now
    console.log(TAG, 'Validating username and assigning role for:', req.body.username);
    var role = 1;
    if (req.body.username.toLowerCase().indexOf('auditor') > -1) {
        role = 3;
    }

    userManager.registerUser(req.body.username, function (err, creds) {
        //console.log('! do i make it here?');
        if (err) {
            req.session.reg_error_msg = 'Failed to register user:' + err.message;
            req.session.registration = null;
            console.error(TAG, req.session.reg_error_msg);
        } else {
            console.log(TAG, 'Registered user:', JSON.stringify(creds));
            req.session.registration = 'Enroll ID: ' + creds.id + '  Secret: ' + creds.secret;
            req.session.reg_error_msg = null;
        }
        res.redirect('/login');
    });
}

/**
 * Handles form posts for enrollment requests.
 * @param req The request containing the enroll form data.
 * @param res The response.
 */
function login(req, res) {
    console.log('site_router.js login() - fired');
    req.session.error_msg = 'Invalid username or password';
    req.session.reg_error_msg = null;

    // Registering the user against a peer can serve as a login checker, for now
    console.log(TAG, 'attempting login for:', req.body.username);
    chaincode_ops.login('WebAppAdmin', req.body.username, req.body.password, function (err, user) {
        if (err) {
            console.error(TAG, 'Login failed:', err);
            return res.redirect('/login');
        } else {
        	console.log('obtained user:' + user.Role);
        	console.log(user.toString());
        	var userJSON = JSON.parse(user.toString());
        	req.session.role = userJSON.role;
        	req.session.username = userJSON.userName;
        	retrieveLCApplications(req, res);
        }
    });
}

function retrieveLCApplications(req, res) {
	chaincode_ops.getAllLCs('WebAppAdmin', function (err, lcList) {
        if (err) {
            console.error(TAG, 'Retrieve all LCs failed:', err);
            return res.render('lcList', {lcApplications : {}});
        } else {
        	console.log('obtained lcList:' + lcList);
        	return res.render('lcList', {lcApplications : JSON.parse(lcList), 
        			role : req.session.role, date : formatDate(new Date()), username : req.session.username});
        }
	})
}

function getDocument(req, res) {
	chaincode_ops.fileView('WebAppAdmin', req.session.shipmentId, req.query.file, function (err, fileStr) {
        if (err) {
            console.log(TAG, 'Get File failed:', err);
        } else {
        	console.log('Got file bytes from blockchain:' + fileStr.toString());
        	var data = base64_decode(fileStr.toString());
            res.writeHead(200, {"Content-Type": "image/png"});
            res.end(data, 'binary');
        }
	})
	console.log("Getting out of get document");
}

