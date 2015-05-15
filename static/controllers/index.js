var IndexCtrl = function(
    $scope, 
    $location, 
    $routeParams, 
    $window,
    picsService) {
  this.picsService_ = picsService;
  this.location_ = $location;
  this.pics = [];

  // For some reason, replace state causes favicon.ico requests to be 
  // sent in Chrome.  
  // TODO: figure out why scrolling causes a stream of http requests.
  $window.onscroll = function() {
    var x = $window.pageXOffset;
    var y = $window.pageYOffset;
    $window.history.replaceState({x:x, y:y}, '');
  }.bind(this);

  $window.scrollTo(0, 0);
  $window.onpopstate = function (ev) {
    if (ev.state != null) {
      $window.scrollTo(ev.state.x, ev.state.y);
    }
  };

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
  picsService.get(startId).then(
    function(pics) {
      if (pics.length > 0) {
        this.nextPageID = pics[pics.length - 1].id;
        this.pics = pics;
      }
    }.bind(this)
  );
}

IndexCtrl.prototype.loadNext = function() {
  this.picsService_.get(this.nextPageID).then(
    function(pics) {
      this.pics = pics;
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
  this.picsService_.create(this.upload.file, this.upload.url)
      .then(function(data) {
        this.pics.unshift(data.data);
      }.bind(this));
};