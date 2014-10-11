(function(){
var IndexCtrl = function($scope, indexPicsService, createPicService) {
  this.pics = [];
  
  this.upload = {
    file: null, 
    url: "",
  };

  indexPicsService.get().then(
    function(data) {
      this.pics = data.data;
    }.bind(this)
  );
  
  this.fileChange = function(elem) {
    if (elem.files.length > 0) {
      this.upload.file = elem.files[0];
    } else {
      this.upload.file = null;
    }
  }.bind(this);
  
  this.createPic = function() {
    createPicService.create(this.upload.file, this.upload.url)
        .then(function(data) {
          this.pics.unshift(data.data);
        }.bind(this));
  }.bind(this);
}

var IndexPicsService = function($http, $q) {
  this.http_ = $http;
  this.q_ = $q;
};

IndexPicsService.prototype.get = function() {
  var deferred = this.q_.defer();
  this.http_.get("/api/findIndexPics").then(
    function(data, status, headers, config) {
      deferred.resolve(data);
    },
    function(data, status, headers, config) {
      console.log(data);
      console.log(status);
    }
  );
  return deferred.promise;
};

var CreatePicService = function($http, $q) {
  this.http_ = $http;
  this.q_ = $q;
};

CreatePicService.prototype.create = function(file, url) {
  var deferred = this.q_.defer();
  
  var data = new FormData();
  data.append("url", url);
  if (file != null) {
    data.append("file", file);
  }
  var postConfig = {
    transformRequest: angular.identity,
    headers: {'Content-Type': undefined},
  };
  this.http_.post("/api/createPic", data, postConfig).then(
    function(data, status, headers, config) {
      deferred.resolve(data);
    },
    function(data, status, headers, config) {
      console.log(data);
      console.log(status);
    }
  );
  return deferred.promise;
};

angular.module('pixur', [])
    .controller("IndexCtrl", IndexCtrl)
    .service("indexPicsService", IndexPicsService)
    .service("createPicService", CreatePicService);
})();