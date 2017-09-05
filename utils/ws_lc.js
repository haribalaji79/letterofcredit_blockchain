'use strict';
/*******************************************************************************
 * Copyright (c) 2015 IBM Corp.
 *
 * All rights reserved.
 *
 * Communication between the CP browser code and this server is sent over web
 * sockets. This file has the code for processing and responding to message sent
 * to the web socket server.
 *
 * Contributors:
 *   David Huffman - Initial implementation
 *   Dale Avery
 *******************************************************************************/

var TAG = 'web_socket:';

// ==================================
// Part 2 - incoming messages, look for type
// ==================================
var async = require('async');
var https = require('https');
var http = require('http');
var peers = null;
var chaincodeHelper;
let requestLib = https;

module.exports.setup = function setup(peerHosts, chaincode_helper, useTLS) {
    if (!(peerHosts && chaincode_helper))
        throw new Error('Web socket handler given incomplete configuration');
    peers = peerHosts;
    chaincodeHelper = chaincode_helper;
    if (!useTLS) requestLib = http;
};

/**
 * A handler for incoming web socket messages.
 * @param socket A socket that we can respond through.
 * @param data An object containing the incoming message data.
 */
module.exports.process_msg = function (socket, data) {

    // Clients must specify the identity to use on their network.  Needs to be someone
    // that this server has enrolled and has the enrollment cert for.
    /*if (!data.user || data.user === '') {
        sendMsg({type: 'error', error: 'user not provided in message'});
        return;
    }*/

    if (data.type == 'createLC') {
    	console.log("Inside createLC:" + data.lc);
        if (data.lc) {
            console.log(TAG, 'creating lc:', data.lc);
            chaincodeHelper.queue.push(function (cb) {
                chaincodeHelper.createLC('WebAppAdmin', data.lc, function (err, result) {
                    if (err != null) {
                        console.error(TAG, 'Error in createLC. No response will be sent. error:', err);
                    }
                    else {
                        console.log(TAG, 'LC created.  No response will be sent. result:', result);
                        sendMsg({type:'createLC', 'result' : result});
                    }

                    cb();
                });
            }, function (err) {
                if (err)
                    console.error(TAG, 'Queued createLC error:', err.message);
                else
                    console.log(TAG, 'Queued createLC job complete');
            });
        }
    } else if (data.type = 'updateStatus') {
    	if (data.req) {
    		chaincodeHelper.queue.push(function (cb) {
                chaincodeHelper.updateStatus('WebAppAdmin', data.req.shipmentId, data.req.status, 
                		data.req.value, function (err, result) {
                    if (err != null) {
                        console.error(TAG, 'Error in updateStatus. No response will be sent. error:', err);
                    }
                    else {
                        console.log(TAG, 'updateStatus success.  No response will be sent. result:', result);
                        sendMsg({type:'updateStatus', 'result' : result, 'statusFlag' : data.req.value, 
                        	'state' : data.req.status});
                    }

                    cb();
                });
            }, function (err) {
                if (err)
                    console.error(TAG, 'Queued createLC error:', err.message);
                else
                    console.log(TAG, 'Queued createLC job complete');
            });
    	}
    }
    
    //call back for getting the blockchain stats, lets get the block height now
    var chain_stats = {};

    function cb_chainstats(e, stats) {
        chain_stats = stats;
        if (stats && stats.height) {
            var list = [];
            for (var i = stats.height - 1; i >= 1; i--) {								//create a list of heights we need
                list.push(i);
                if (list.length >= 8) break;
            }

            list.reverse();
            async.eachLimit(list, 1, function (key, cb) {							//iter through each one, and send it
                //get chainstats through REST API
                var options = {
                    host: peers[0].api_host,
                    port: peers[0].api_port,
                    path: '/chain/blocks/' + key,
                    method: 'GET'
                };

                function success(statusCode, headers, stats) {
                    stats = JSON.parse(stats);
                    stats.height = key;
                    sendMsg({msg: 'chainstats', e: e, chainstats: chain_stats, blockstats: stats});
                    cb(null);
                }

                function failure(statusCode, headers, msg) {
                    console.log('chainstats block ' + key);
                    console.log('status code: ' + statusCode);
                    console.log('headers: ' + headers);
                    console.log('message: ' + msg);
                    cb(null);
                }

                var request = requestLib.request(options, function (resp) {
                    var str = '', chunks = 0;
                    resp.setEncoding('utf8');
                    resp.on('data', function (chunk) {															//merge chunks of request
                        str += chunk;
                        chunks++;
                    });
                    resp.on('end', function () {
                        if (resp.statusCode == 204 || resp.statusCode >= 200 && resp.statusCode <= 399) {
                            success(resp.statusCode, resp.headers, str);
                        }
                        else {
                            failure(resp.statusCode, resp.headers, str);
                        }
                    });
                });

                request.on('error', function (e) {																//handle error event
                    failure(500, null, e);
                });

                request.setTimeout(20000);
                request.on('timeout', function () {																//handle time out event
                    failure(408, null, 'Request timed out');
                });

                request.end();
            }, function () {
            });
        }
    }

    /**
     * Send a response back to the client.
     * @param json The content of the response.
     */
    function sendMsg(json) {
        if (socket) {
            try {
                socket.send(JSON.stringify(json));
            }
            catch (error) {
                console.error('Error sending response to client:', error.message);
            }
        }
    }
};