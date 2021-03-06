/*
 * Cryptexly - Copyleft of Xavier D. Johnson.
 * me at xavierdjohnson dot com
 * http://www.xavierdjohnson.net/
 *
 * See LICENSE.
 */

function Cryptexly(on_req_executed) {
    this.token = window.localStorage.arcToken;
    this.token_time = window.localStorage.arcTokenTime;
    this.store = null;
    this.config = null;
    this.records = null;
}

Cryptexly.prototype.IsLogged = function() {
    return ( this.token != null );
}

Cryptexly.prototype.HasStore = function() {
    return this.IsLogged() && ( this.store != null );
}

Cryptexly.prototype.Api = function( method, path, data, success, error, raw ) {
    var cryptexly = this;
    return $.ajax({
        type: method,
        url: path,
        beforeSend: function (xhr) {
            if( cryptexly.token != null ) {
                xhr.setRequestHeader('Authorization', 'Bearer: ' + cryptexly.token);
            }
        },
        success: success,
        error: error,
        data: data ? JSON.stringify(data) : null,
        contentType: "application/json",
        dataType: raw ? undefined : 'json',
        cache: false,
        timeout: 60 * 60 * 1000
    });
}

Cryptexly.prototype.ApiStream = function( method, path, meta, blob, success, error ) {
    /*
     * I wish I cold use this, but on mobile FormData is not
     * well supported yet :(

      var form_data = new FormData();
      var file = new File(
          [blob],
          'data',
          { type: 'application/octet-stream' }
      );
     
      form_data.append( "meta", JSON.stringify(meta) );
      form_data.append( "data", file );
    */
    var cryptexly = this;
    var boundary = "AJAX-----------------------" + (new Date).getTime();
    var ctype = "multipart/form-data; boundary=" + boundary
    var CRLF = "\r\n";
    var form_data = "--" + boundary + CRLF +
                    "Content-Disposition: form-data; name=\"meta\"" + CRLF + CRLF +
                    JSON.stringify(meta) + CRLF +
                    "--" + boundary + CRLF +
                    "Content-Disposition: form-data; name=\"data\"; filename=\"data\"" + CRLF +
                    "Content-Type: application/octet-stream" + CRLF + CRLF +
                    blob + CRLF + 
                    "--" + boundary + "--" + CRLF;

    return $.ajax({
        type: method,
        url: path,
        beforeSend: function (xhr) {
            if( cryptexly.token != null ) {
                xhr.setRequestHeader('Authorization', 'Bearer: ' + cryptexly.token);
            }
        },
        success: success,
        error: error,
        cache: false,
        processData: false,
        contentType: ctype,
        data: form_data,
        timeout: 60 * 60 * 1000
    });
}

Cryptexly.prototype.Logout = function() {
    console.log( "Logging out, deleting token " + this.token );
    this.token = null;
    this.token_time = null;
}

Cryptexly.prototype.Login = function(username, password, persist, success, error) {
    console.log( "Logging in ..." );
    
    var v = this;
    var login = { username: username, password: password };

    this.Api( 'POST', '/auth', login, function(resp) {
        if( resp.token != null ) {
            v.token = resp.token;
            v.token_time = Date.now();
            if( persist ) {
                console.log( "Saving token to browser storage." );
                window.localStorage.arcToken = v.token;
                window.localStorage.arcTokenTime = v.token_time;
            }

            v.Api( 'GET', '/api/config', null, function(resp) {
                v.config = resp;
            }, error);

            success(resp.token);
        } else {
            error(resp);
        }
    },
    error);
}

Cryptexly.prototype.Status = function( success, error ) {
    return this.Api( 'GET', '/api/status', null, success, error );
};

Cryptexly.prototype.ClearEvents = function( success, error ) {
    return this.Api( 'GET', '/api/events/clear', null, success, error );
};

Cryptexly.prototype.Stores = function(success, error) {
    this.Api( 'GET', '/api/stores', null, success, error );
}

Cryptexly.prototype.Records = function( store, success, error ) {
    console.log( "Loading store-" + store.id + " records ..." );
    var cryptexly = this;
    this.Api( 'GET', '/api/store/' + store.id + '/records', null, function(records){
        cryptexly.records = records;
        success();
    }, error );
}

Cryptexly.prototype.GetStore = function(id, success, error) {
    this.Api( 'GET', '/api/store/' + id, null, success, error );
}

Cryptexly.prototype.DeleteStore = function(id, success, error) {
    this.Api( 'DELETE', '/api/store/' + id, null, success, error );
}

Cryptexly.prototype.AddStore = function( title, success, error ) {
    console.log( "Creating store '" + title + "'" );
    var record = {
        'Title': title,
    };
    this.Api( 'POST', '/api/stores/', record, success, error );
}

Cryptexly.prototype.UpdateStore = function( id, title, success, error ) {
    console.log( "Updating store " + id + " to '" + title + "'" );
    var store = {
        'ID': id,
        'Title': title,
    };
    this.Api( 'PUT', '/api/store/' + id, store, success, error );
}

Cryptexly.prototype.SetStore = function( id, success, error ) {
    console.log( "Selecting store '" + id + "'" );
    var cryptexly = this;
    this.GetStore( id, function(s){
        cryptexly.store = s;
        cryptexly.Records( s, success, error );
    },
    error);
}

Cryptexly.prototype.GetRecordBuffer = function( record_id, success, error ) {
    var path = '/api/store/' + this.store.id + '/record/' + record_id + '/buffer';
    return this.Api( 'GET', path, null, success, error, true );
};

Cryptexly.prototype.AddRecordTo = function( record, data, store_id, success, error ) {
    return this.ApiStream( 'POST', '/api/store/' + store_id + '/records', record, data, success, error );
}

Cryptexly.prototype.AddRecord = function( record, data, success, error ) {
    if( this.HasStore() == false ) {
        return error("No store has been selected.");
    }
    return this.AddRecordTo(record, data, this.store.id, success, error );
}

Cryptexly.prototype.UpdateRecord = function( record, data, success, error) {
    if( this.HasStore() == false ) {
        return error("No store has been selected.");
    }

    return this.ApiStream( 'PUT', '/api/store/' + this.store.id + '/record/' + record.id, record, data, success, error );
}

Cryptexly.prototype.DeleteRecord = function( record, success, error ) {
    if( this.HasStore() == false ) {
        return error("No store has been selected.");
    }
    this.Api( 'DELETE', '/api/store/' + this.store.id + '/record/' + record.id, null, success, error );
}
