
var ViewerCtrl = function($scope, $routeParams, $window, picsService, authService) {
  this.isImage = false;
  this.isVideo = false;
  
  this.canDelete = false;
  
  this.picId = $routeParams.picId;
  this.pic = null;
  this.picTags = [];
  
  this.picsService_ = picsService;
  // TODO: use the location api instead of this hack.
  this.window_ = $window;

  authService.getXsrfToken().then(function() {
    return picsService.getSingle(this.picId)
  }.bind(this)).then(function(details) {
    this.pic = details.pic;
    this.picTags = details.pic_tags;
    this.isVideo = this.pic.type == "WEBM";
    this.isImage = this.pic.type != "WEBM";

    picsService.incrementViewCount(this.picId);
  }.bind(this));
}

ViewerCtrl.prototype.deletePic = function() {
  this.picsService_.deletePic(this.picId).then(
    function(f) {
      this.window_.history.back()
    }.bind(this),
    function(err) {
      // TODO: actually return a better error
      alert(angular.toJson(err, 2));
      console.error(err);
    });
}
