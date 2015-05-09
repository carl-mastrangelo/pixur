
var ViewerCtrl = function($scope, $routeParams, $window, picsService) {
  this.isImage = false;
  this.isVideo = false;
  
  this.canDelete = false;
  
  this.picId = $routeParams.picId;
  this.pic = null;
  this.picTags = [];
  
  this.picsService_ = picsService;
  // TODO: use the location api instead of this hack.
  this.window_ = $window;

  picsService.getSingle(this.picId).then(
    function(details) {
      this.pic = details.pic;
      this.picTags = details.pic_tags;
      this.isVideo = this.pic.type == "WEBM";
      this.isImage = this.pic.type != "WEBM";
    }.bind(this)
  );
}

ViewerCtrl.prototype.deletePic = function() {
  this.picsService_.deletePic(this.picId).then(
    function() {
      this.window_.history.back()
    }.bind(this),
    function(err) {
      // TODO: actually return a better error
      alert(err);
      console.error(err);
    });
}

