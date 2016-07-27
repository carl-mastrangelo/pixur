var AuthService = function($http, $q, $cookies, $window) {
  this.http_ = $http;
  this.q_ = $q;
  this.cookies_ = $cookies;
  this.window_ = $window;
};

AuthService.prototype.getXsrfToken = function() {
  var deferred = this.q_.defer();
  
  var token = this.cookies_.get("XSRF-TOKEN");
  if (token && token.length > 0) {
    deferred.resolve(true);
  } else {
    this.http_.post("/api/getXsrfToken").then(
      function(res) {
        deferred.resolve(true);
      },
      function(error) {
        deferred.reject(error);
      }
    );
  }

  return deferred.promise;
}


AuthService.prototype.createUser = function(ident, secret) {
  var deferred = this.q_.defer();
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
  return this.http_.post("/api/createUser", params, httpConfig);
};

AuthService.prototype.getAuth = function() {
	var item = this.window_.localStorage.getItem("auth");
	if (item) {
		return this.window_.JSON.parse(item);
	}
	return null;
};

AuthService.prototype.loginUser = function(ident, secret) {
  var deferred = this.q_.defer();
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
  return this.http_.post("/api/getSession", params, httpConfig).then(function(res) {
  	var s = {
  		"user_id": res.data.jwtPayload.sub,
  		"ident": ident
  	};
  	this.window_.localStorage.setItem("auth", this.window_.JSON.stringify(s));
  }.bind(this));
};


// copy of PicsService.postTransform
AuthService.postTransform = function(o) {
  var str = [];
  for(var p in o)
  str.push(encodeURIComponent(p) + "=" + encodeURIComponent(o[p]));
  return str.join("&");
};
