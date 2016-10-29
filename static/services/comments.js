var CommentsService = function($http, $q, authService) {
  this.http_ = $http;
  this.q_ = $q;
  this.authService_ = authService;
};

CommentsService.prototype.addComment = function(picId, commentParentId, text) {
  var deferred = this.q_.defer();
  var params = {
      pic_id: picId,
      comment_parent_id: commentParentId,
      text: text
  }
  var httpConfig = {
    "headers":  {
      "Content-Type": "application/x-www-form-urlencoded"
    },
    "transformRequest": CommentsService.postTransform
  };
  
  this.authService_.getXsrfToken().then(function() {
  	return this.authService_.getAuth();
  }.bind(this)).then(function() {
  	return this.http_.post("/api/addPicComment", params, httpConfig);
  }.bind(this)).then(
    function(res) {
      deferred.resolve(res.data);
    },
    function(error) {
      deferred.reject(error);
    }
  );

  return deferred.promise;
}


// Must be in post order
CommentsService.prototype.buildTree = function(comments) {
	var map = {};
	map[0] = {};
	for (var i = comments.length - 1; i >= 0; i--) {
		map[comments[i].commentId] = comments[i];
		if (!("children" in map[comments[i].commentParentId])) {
			map[comments[i].commentParentId].children = [];
		}
		map[comments[i].commentParentId].children.push(comments[i])
	}
	return map[0];
}

CommentsService.postTransform = function(o) {
  var str = [];
  for(var p in o)
  str.push(encodeURIComponent(p) + "=" + encodeURIComponent(o[p]));
  return str.join("&");
};
