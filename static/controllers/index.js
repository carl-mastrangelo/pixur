var IndexCtrl = function(
    $scope, 
    $location, 
    $routeParams, 
    $window,
    picsService,
    authService) {
  this.picsService_ = picsService;
  this.authService_ = authService;
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
  
  this.auth = authService.getIdent(); 
  this.nextPageID = null;
  this.prevPageID = null;
  
  this.upload = {
    file: null, 
    url: "",
  };
  var startId = "";
  var haveStartId = $routeParams.picId !== undefined;
  if (haveStartId) {
    startId = $routeParams.picId;
  }

  picsService.getNextIndexPics(startId).then(function(pics) {
    if (pics.length >= 1) {
      this.pics = pics;
    }
    if (pics.length >= 2) {
      this.nextPageID = pics[pics.length - 1].id;
    } else {
      this.nextPageID = null;
    }
  }.bind(this));
  // If start id is not specified, then loading the previous pics
  // searches backwards from 0, which makes the index wrap around.
  if (haveStartId) {
    picsService.getPreviousIndexPics(startId).then(
      function(pics) {     
        if (pics.length >= 2) {
          this.prevPageID = pics[pics.length - 1].id;
        } else {
          // We always get back the pic Id we asked for, or nothing.
          // If only 1 or 0 pics come back, we reached the edge.
          this.prevPageID = null;
        }
      }.bind(this)
    );
  }
}

IndexCtrl.prototype.logOut = function() {
  this.authService_.logoutUser().catch(err => {
  	console.warn(err);
  });
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
        this.pics.unshift(data.pic);
      }.bind(this));
};
