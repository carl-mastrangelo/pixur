
var ViewerCtrl = function($scope, $routeParams, picsService) {
  this.isImage = false;
  this.isVideo = false;
  
  this.picId = $routeParams.picId;
  this.pic = null;
  this.picTags = [];

  picsService.getSingle(this.picId).then(
    function(details) {
      this.pic = details.pic;
      this.picTags = details.pic_tags;
      this.isVideo = this.pic.type == "WEBM";
      this.isImage = this.pic.type != "WEBM";
    }.bind(this)
  );
}
