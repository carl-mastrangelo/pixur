
var LoginCtrl = function($scope, $http, $window, authService) {
	this.loginIdent = null;
	this.loginSecret = null;
  this.createIdent = null;
	this.createSecret = null;
	this.http_ = $http;
	this.authService_ = authService; 
	this.window_ = $window;
	
	this.errorText = null;
}

LoginCtrl.prototype.createUser = function() {
	this.authService_.getXsrfToken().then(function() {
    return this.authService_.createUser(this.createIdent, this.createSecret);
  }.bind(this)).then(function (res) {
  	return this.authService_.loginUser(this.createIdent, this.createSecret);
  }.bind(this)).then(function (res) {
  	this.window_.history.back();
  }.bind(this)).catch(function(e) {
  	this.errorText = e;
  	console.error(e);
  }.bind(this));
};

LoginCtrl.prototype.loginUser = function() {
	this.authService_.getXsrfToken().then(function (res) {
  	return this.authService_.loginUser(this.loginIdent, this.loginSecret);
  }.bind(this)).then(function (res) {
  	this.window_.history.back();
  }.bind(this)).catch(function(e) {
  	this.errorText = e;
  	console.error(e);
  }.bind(this));
};


