(function () {
  const navigation = document.querySelector(".navigation");
  const rootPrefix = navigation.dataset.rootPrefix || "";

  document.getElementById("show-projects").addEventListener("click", function () {
    window.location.href = rootPrefix + "index.html";
  });
  document.getElementById("show-holders").addEventListener("click", function () {
    window.location.href = rootPrefix + "holders.html";
  });
  document.getElementById("show-trends").addEventListener("click", function () {
    window.location.href = rootPrefix + "trends.html";
  });
})();
