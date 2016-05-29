var AuthService = function($http, $q, $cookies) {
  this.http_ = $http;
  this.q_ = $q;
  this.cookies_ = $cookies;
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

