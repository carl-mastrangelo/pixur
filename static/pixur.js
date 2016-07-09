
angular.module('pixur', [
      "ngRoute", "ngCookies"
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
          .when("/exp/qv/:picId", {
            templateUrl: "static/quickviewer.html",
            controller: "QuickViewerCtrl",
            controllerAs: "ctrl"
          })
          .when("/u/login", {
            templateUrl: "static/login.html",
            controller: "LoginCtrl",
            controllerAs: "ctrl"
          })
      ;
    })
    .controller("IndexCtrl", IndexCtrl)
    .controller("ViewerCtrl", ViewerCtrl)
    .controller("QuickViewerCtrl", QuickViewerCtrl)
    .controller("LoginCtrl", LoginCtrl)
    .service("picsService", PicsService)
    .service("authService", AuthService);
