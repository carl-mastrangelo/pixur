var IndexCtrl = function(
    $scope, 
    $location, 
    $routeParams, 
    picsService) {
  this.picsService_ = picsService;
  this.location_ = $location;
  this.pics = [];

  this.nextPageID = null;
  this.prevPageID = null;
  
  window.onpopstate = function (ev) {
    console.log(ev);
  }
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
  this.picsService_.get(this.nextPageID).then(
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
  this.picsService_.create(this.upload.file, this.upload.url)
      .then(function(data) {
        this.pics.unshift(data.data);
      }.bind(this));
};