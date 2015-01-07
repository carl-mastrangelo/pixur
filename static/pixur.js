(function(){
var IndexCtrl = function(
    $scope, 
    $location, 
    $routeParams, 
    indexPicsService, 
    createPicService) {
  this.indexPicsService_ = indexPicsService;
  this.createPicService_ = createPicService;
  this.location_ = $location;
  this.pics = [];

  this.nextPageID = null;
  this.prevPageID = null;

  this.upload = {
    file: null, 
    url: "",
  };
  var startId = 0;
  if ($routeParams.picId) {
    startId = $routeParams.picId;
  }

  // Initial Load
  indexPicsService.get(startId).then(
    function(data) {
      var pics = data.data;
      if (pics.length > 0) {
        this.nextPageID = pics[pics.length - 1].id;
        this.pics = pics;
      }
    }.bind(this)
  );
}

IndexCtrl.prototype.loadNext = function() {
  this.indexPicsService_.get(this.nextPageID).then(
    function(data) {
      this.pics = data.data;
      if (this.pics.length > 0) {
        this.nextPageID = this.pics[this.pics.length - 1].id
      }
    }.bind(this)
  );
}

IndexCtrl.prototype.fileChange = function(elem) {
  if (elem.files.length > 0) {
    this.upload.file = elem.files[0];
  } else {
    this.upload.file = null;
  }
};

IndexCtrl.prototype.createPic = function() {
  this.createPicService_.create(this.upload.file, this.upload.url)
      .then(function(data) {
        this.pics.unshift(data.data);
      }.bind(this));
};

var ViewerCtrl = function($scope, $routeParams, indexPicsService) {
  this.isImage = false;
  this.isVideo = false;
  
  this.picId = $routeParams.picId;
  this.pic = null;
  
  // TODO: hack, poor performance, replace with something less awful
  // Initial Load
  indexPicsService.get(this.picId).then(
    function(data) {
      var pics = data.data;
      if (pics.length > 0) {
        this.pic = pics[0];
        this.isVideo = this.pic.type == "WEBM";
        this.isImage = this.pic.type != "WEBM";
      }
    }.bind(this)
  );
}

var IndexPicsService = function($http, $q, $cacheFactory) {
  this.http_ = $http;
  this.q_ = $q;
  
  this.cache = $cacheFactory.get("IndexPicsService");
  if (!this.cache) {
   this.cache = $cacheFactory("IndexPicsService", {
    capacity: 10
   });
  }
};

IndexPicsService.prototype.get = function(startID) {
  var deferred = this.q_.defer();
  var cache;
  // Only cache if startID is not 0, basically if not the home page.
  if (startID) {
    cache = this.cache
  }
  var httpConfig = {
    cache:cache
  };
  if (startID) {
    httpConfig.params = {
      start_pic_id: startID
    };
  }
  this.http_.get("/api/findIndexPics", httpConfig).then(
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

var ScrollDirective = function($window) {
  return function(scope, element, attrs) {
    angular.element($window).bind("scroll", function() {
      //console.log(this);
    });
  };
};

angular.module('pixur', [
      "ngRoute"
    ])
    .config(function($locationProvider) {
      $locationProvider.html5Mode(true);
    })
    .config(function($routeProvider) {
      $routeProvider
          .when("/", {
            templateUrl: "static/index_body.html",
            controller: "IndexCtrl",
            controllerAs: "ctrl"
          })
          .when("/i/:picId?", {
            templateUrl: "static/index_body.html",
            controller: "IndexCtrl",
            controllerAs: "ctrl"
          })
          .when("/p/:picId", {
            templateUrl: "static/viewer.html",
            controller: "ViewerCtrl",
            controllerAs: "ctrl"
          })
      ;
    })
    .controller("IndexCtrl", IndexCtrl)
    .controller("ViewerCtrl", ViewerCtrl)
    .service("indexPicsService", IndexPicsService)
    .service("createPicService", CreatePicService)
    .directive("scroll", ScrollDirective);
})();