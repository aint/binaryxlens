(function () {
  const indexPath = "index.html";
  const sitePath = "site/";
  const holdersPage = sitePath + "holders.html";
  const trendsPage = sitePath + "trends.html";
  const marketsPage = sitePath + "markets.html";
  const navigation = document.querySelector(".navigation");
  const rootPrefix = navigation.dataset.rootPrefix || (window.location.pathname.includes("/site/") ? "../" : "");

  document.getElementById("show-projects").addEventListener("click", function () {
    window.location.href = rootPrefix + indexPath;
  });
  document.getElementById("show-holders").addEventListener("click", function () {
    window.location.href = rootPrefix + holdersPage;
  });
  document.getElementById("show-trends").addEventListener("click", function () {
    window.location.href = rootPrefix + trendsPage;
  });
  document.getElementById("show-markets").addEventListener("click", function () {
    window.location.href = rootPrefix + "site/markets.html";
  });
})();
