var QuickViewerCtrl = function($scope, $q, $routeParams, $location, picsService) {
  this.isImage = false;
  this.isVideo = false;
  
  this.canDelete = false;
  
  this.picId = $routeParams.picId;
  this.pic = null;
  this.pics = [];
  this.nextPageID = $routeParams.picId;
  this.picsService_ = picsService;
  this.q_ = $q;
  // TODO: use the location api instead of this hack.
  this.location_ = $location;

  this.loadNext();
};

QuickViewerCtrl.prototype.loadNext = function() {
  var deferred = this.q_.defer();
  if(!this.pics.length) {
    this.picsService_.get(this.nextPageID).then(
      function(pics) {
        if (pics.length > 0) {
          this.nextPageID = pics[pics.length - 1].id;
          this.pics = pics;
          deferred.resolve(true);
        }
      }.bind(this),
      function (err) {
         deferred.reject(err);
      }
    );
  } else {
    deferred.resolve(true);
  }
  return deferred.promise.then(function () {
    var p = this.pics.shift();
    this.picId = p.id;
    this.picsService_.getSingle(this.picId).then(
      function(details) {
        this.pic = details.pic;
        this.isVideo = this.pic.type == "WEBM";
        this.isImage = this.pic.type != "WEBM";
      }.bind(this)
    );
  }.bind(this));
};


QuickViewerCtrl.prototype.keydown = function(ev) {
  if(ev.which == 119) { //w
  } else if(ev.which == 97) { //a
  } else if(ev.which == 115) {
    //s
    this.picsService_.deletePic(this.picId, "downvote").then(
      function(){},
      function(err){ console.log(err); }
    );
    this.loadNext();
  } else if(ev.which == 100) {//d
    this.loadNext();
  }
};

