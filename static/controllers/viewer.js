
var ViewerCtrl = function($scope, $routeParams, $window, picsService, commentsService, 
		authService) {
  this.isImage = false;
  this.isVideo = false;
  
  this.canDelete = false;
  
  this.picId = $routeParams.picId;
  this.pic = null;
  this.picTags = [];

  this.picsService_ = picsService;
  this.commentsService_ = commentsService;
  // TODO: use the location api instead of this hack.
  this.window_ = $window;
  
  //this.topcomment = "";

  authService.getXsrfToken().then(function() {
    return picsService.getSingle(this.picId)
  }.bind(this)).then(function(details) {
    this.pic = details.pic;
    this.picTags = details.pic_tags;
    this.isVideo = this.pic.type == "WEBM";
    this.isImage = this.pic.type != "WEBM";
    this.picComments = commentsService.buildTree(details.picCommentTree.comment || []).children;
		
    picsService.incrementViewCount(this.picId);
  }.bind(this));
}

ViewerCtrl.prototype.addComment = function(commentParent, text) {
	var commentParentId = 0;
	if (commentParent) {
		commentParentId = commentParent.commentId;
	}
  this.commentsService_.addComment(this.picId, commentParentId, text).then(
    function(f) {
			this.picsService_.getSingle(this.picId).then(function(details) {
				this.picComments = this.commentsService_.buildTree(details.picCommentTree.comment).children;
			}.bind(this));
    }.bind(this),
    function(err) {
      alert(angular.toJson(err, 2));
      console.error(err);
    });
}

ViewerCtrl.prototype.deletePic = function() {
  this.picsService_.deletePic(this.picId).then(
    function(f) {
      this.window_.history.back();
    }.bind(this),
    function(err) {
      // TODO: actually return a better error
      alert(angular.toJson(err, 2));
      console.error(err);
    });
}

ViewerCtrl.prototype.voteUp = function() {
  this.picsService_.vote(this.picId, "UP").then(
    function(f) {}.bind(this),
    function(err) {
      // TODO: actually return a better error
      alert(angular.toJson(err, 2));
      console.error(err);
    });
}

ViewerCtrl.prototype.voteDown = function() {
  this.picsService_.vote(this.picId, "DOWN").then(
    function(f) {}.bind(this),
    function(err) {
      // TODO: actually return a better error
      alert(angular.toJson(err, 2));
      console.error(err);
    });
}

ViewerCtrl.prototype.voteNeutral = function() {
  this.picsService_.vote(this.picId, "NEUTRAL").then(
    function(f) {}.bind(this),
    function(err) {
      // TODO: actually return a better error
      alert(angular.toJson(err, 2));
      console.error(err);
    });
}
