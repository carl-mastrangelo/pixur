(function(){
var IndexCtrl = function($scope, indexPicsService) {
  this.pics = [];
  
  
  indexPicsService.get().then(
    function(data) {
      this.pics = data.data;
    }.bind(this)
  );
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

angular.module('pixur', [])
    .controller("IndexCtrl", IndexCtrl)
    .service("indexPicsService", IndexPicsService);
})();