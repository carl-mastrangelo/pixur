
var ViewerCtrl = function($scope, $routeParams, picsService) {
  this.isImage = false;
  this.isVideo = false;
  
  this.picId = $routeParams.picId;
  this.pic = null;
  
  // TODO: hack, poor performance, replace with something less awful
  // Initial Load
  picsService.getSingle(this.picId).then(
    function(pic) {
      this.pic = pic;
      this.isVideo = pic.type == "WEBM";
      this.isImage = pic.type != "WEBM";
    }.bind(this)
  );
}
