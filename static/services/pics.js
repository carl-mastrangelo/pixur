var PicsService = function($http, $q, $cacheFactory) {
  this.http_ = $http;
  this.q_ = $q;
  
  this.indexCache = $cacheFactory.get("PicsService");
  if (!this.indexCache) {
   this.indexCache = $cacheFactory("PicsService", {
    capacity: 20
   });
   
   this.picCache = $cacheFactory.get("PicsService-pics");
   if (!this.picCache) {
     this.picCache = $cacheFactory("PicsService-pics", {
      capacity: 122 // 2x Default page size plus one for good measure
     });
   }
  }
};

PicsService.prototype.getSingle = function(picId) {
  var deferred = this.q_.defer();
  var httpConfig = {
    params: {
      pic_id: picId
    }
  };
  this.http_.get("/api/lookupPicDetails", httpConfig).then(
    function(res) {
      deferred.resolve(res.data);
    },
    function(error) {
      deferred.reject(error);
    }
  );
  
  return deferred.promise;
}

PicsService.prototype.incrementViewCount = function(picId) {
  var deferred = this.q_.defer();
  var httpConfig = {
    params: {
      pic_id: picId
    }
  };
  this.http_.get("/api/incrementPicViewCount", httpConfig).then(
    function(res) {
      deferred.resolve(res.data);
    },
    function(error) {
      deferred.reject(error);
    }
  );

  return deferred.promise;
}

PicsService.prototype.deletePic = function(picId, details) {
  var deferred = this.q_.defer();
  var httpConfig = {
    params: {
      pic_id: picId,
      details: details
      // TODO: add pending deletion time
    }
  };
  this.http_.get("/api/softDeletePic", httpConfig).then(
    function(res) {
      this.indexCache.removeAll();
      this.picCache.removeAll();
      deferred.resolve(res.data);
    }.bind(this),
    function(error) {
      deferred.reject(error);
    }
  );
  return deferred.promise;
}

PicsService.prototype.get = function(startID) {
  var deferred = this.q_.defer();
  var indexCache;
  var picCache = this.picCache;
  // Only cache if startID is not 0, basically if not the home page.
  if (startID) {
    indexCache = this.indexCache
  }
  var httpConfig = {
    cache:indexCache
  };
  if (startID) {
    httpConfig.params = {
      start_pic_id: startID
    };
  }
  this.http_.get("/api/findNextIndexPics", httpConfig).then(
    function(res, status, headers, config) {
      res.data.forEach(function(pic){
        picCache.put(pic.id, pic);
      });
      deferred.resolve(res.data);
      
      if (res.data.length) {
        httpConfig.params = {
          start_pic_id: res.data[res.data.length -1].id
        };
        this.http_.get("/api/findNextIndexPics", httpConfig).then(
        function(res) {
          res.data.forEach(function(pic){
            picCache.put(pic.id, pic);
          });
        }.bind(this),
        function(error) {
          console.error("Failure to prefetch: ", error)
        });
      }

    }.bind(this),
    function(error) {
      deferred.reject(error);
    }
  );
  return deferred.promise;
};

PicsService.prototype.create = function(file, url) {
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

