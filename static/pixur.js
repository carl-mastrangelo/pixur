
angular.module('pixur', [
      "ngRoute"
    ])
    .config(function($locationProvider) {
      $locationProvider.html5Mode(true);
    })
    .config(function($routeProvider) {
      $routeProvider
          .when("/", {
            templateUrl: "static/index_body.html",
            controller: "IndexCtrl",
            controllerAs: "ctrl"
          })
          .when("/i/:picId?", {
            templateUrl: "static/index_body.html",
            controller: "IndexCtrl",
            controllerAs: "ctrl"
          })
          .when("/p/:picId", {
            templateUrl: "static/viewer.html",
            controller: "ViewerCtrl",
            controllerAs: "ctrl"
          })
      ;
    })
    .controller("IndexCtrl", IndexCtrl)
    .controller("ViewerCtrl", ViewerCtrl)
    .service("picsService", PicsService);
