(function () {
  const navigation = document.querySelector(".navigation");
  const rootPrefix = navigation.dataset.rootPrefix || "";

  document.getElementById("show-projects").addEventListener("click", function () {
    window.location.href = rootPrefix + "index.html";
  });
  document.getElementById("show-holders").addEventListener("click", function () {
    window.location.href = rootPrefix + "site/holders.html";
  });
  document.getElementById("show-trends").addEventListener("click", function () {
    window.location.href = rootPrefix + "site/trends.html";
  });
  document.getElementById("show-markets").addEventListener("click", function () {
    window.location.href = rootPrefix + "site/markets.html";
  });
})();
