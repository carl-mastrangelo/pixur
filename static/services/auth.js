var AuthService = function($http, $q, $cookies, $window) {
  this.http_ = $http;
  this.q_ = $q;
  this.cookies_ = $cookies;
  this.window_ = $window;
  
  this.getAuthPromise = null;
  this.getXsrfTokenPromise = null;
};

AuthService.prototype.getXsrfToken = function() {
	if (this.getXsrfTokenPromise) {
		return this.getXsrfTokenPromise;
	}
  var deferred = this.q_.defer();
  this.getXsrfTokenPromise = deferred.promise;
  
  var token = this.cookies_.get("XSRF-TOKEN");
  if (token && token.length > 0) {
    deferred.resolve(true);
    this.getXsrfTokenPromise = null;
  } else {
    this.http_.post("/api/getXsrfToken").then(
      function(res) {
        deferred.resolve(true);
        this.getXsrfTokenPromise = null;
      }.bind(this),
      function(error) {
        deferred.reject(error);
        this.getXsrfTokenPromise = null;
      }.bind(this)
    );
  }

  return deferred.promise;
};


AuthService.prototype.createUser = function(ident, secret) {
  var params = {
    "ident": ident,
    "secret": secret
  }
  var httpConfig = {
    "headers":  {
      "Content-Type": "application/x-www-form-urlencoded"
    },
    "transformRequest": AuthService.postTransform
  };
  return this.getXsrfToken().then(function() {
  	return this.http_.post("/api/createUser", params, httpConfig);
  }.bind(this));
};

AuthService.prototype.getIdent = function() {
	var ident = this.window_.localStorage.getItem("ident");
	if (ident) {
		return this.window_.JSON.parse(ident);
	}
	return null; 
};

AuthService.prototype.getAuth = function() {
	if (this.getAuthPromise) {
		return this.getAuthPromise;
	}
	var deferred = this.q_.defer();
	this.getAuthPromise = deferred.promise;
	var authRaw = this.window_.localStorage.getItem("auth");
	var auth = null;
	if (authRaw) {
		 auth = this.window_.JSON.parse(authRaw);
	}
	if (!auth || (new Date() >= new Date(auth.notAfter))) {
		this.refreshToken().then(function() {
			deferred.resolve(true);
			this.getAuthPromise = null;
		}.bind(this));
	} else {
		deferred.resolve(true);
		this.getAuthPromise = null;
	}
	return deferred.promise;
};

AuthService.prototype.loginUser = function(ident, secret) {
  var params = {
    "ident": ident,
    "secret": secret
  }
  var httpConfig = {
    "headers":  {
      "Content-Type": "application/x-www-form-urlencoded"
    },
    "transformRequest": AuthService.postTransform
  };
  return this.getXsrfToken().then(function() {
  	return this.http_.post("/api/getRefreshToken", params, httpConfig);
  }.bind(this)).then(function(res) {
  	var identTuple = {
  		"user_id": res.data.refreshPayload.subject,
  		"ident": ident
  	};
  	this.window_.localStorage.setItem("ident", this.window_.JSON.stringify(identTuple));
  	this.window_.localStorage.setItem("refresh", 
  		this.window_.JSON.stringify(res.data.refreshPayload));
  	this.window_.localStorage.setItem("auth", 
  		this.window_.JSON.stringify(res.data.authPayload));
  }.bind(this));
};

AuthService.prototype.logoutUser = function() {
  var httpConfig = {
    "headers":  {
      "Content-Type": "application/x-www-form-urlencoded"
    },
    "transformRequest": AuthService.postTransform
  };
  return this.getXsrfToken().then(() => {
  	return this.http_.post("/api/deleteToken", {}, httpConfig);
  }).then(res => {
  	var ls = this.window_.localStorage;
  	ls.removeItem("ident");
  	ls.removeItem("refresh");
  	ls.removeItem("auth");
  });
};

AuthService.prototype.refreshToken = function() {
  var params = {};
  var httpConfig = {
    "headers":  {
      "Content-Type": "application/x-www-form-urlencoded"
    },
    "transformRequest": AuthService.postTransform
  };
  return this.getXsrfToken().then(function() {
    return this.http_.post("/api/getRefreshToken", params, httpConfig);
  }.bind(this)).then(function(res) {
  	this.window_.localStorage.setItem("refresh", 
  		this.window_.JSON.stringify(res.data.refreshPayload));
  	this.window_.localStorage.setItem("auth", 
  		this.window_.JSON.stringify(res.data.authPayload));
  }.bind(this));
};

// copy of PicsService.postTransform
AuthService.postTransform = function(o) {
  var str = [];
  for(var p in o)
  str.push(encodeURIComponent(p) + "=" + encodeURIComponent(o[p]));
  return str.join("&");
};
