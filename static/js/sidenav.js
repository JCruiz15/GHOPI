function toggleSideNav() {

  sidenav = document.getElementById("sideNav");
  main = document.getElementById("main");
  icon = document.getElementById("sidenav-icon");

  if (sidenav.style.width == "300px") {
    sidenav.style.width = "0px";
    document.getElementById("opensidenav").className="open"
    icon.innerText = "arrow_forward_ios";
    main.style.marginLeft = "0px";

  } else {
    sidenav.style.width = "300px";
    document.getElementById("opensidenav").className="close"
    icon.innerText = "arrow_back_ios";
    main.style.marginLeft = "300px";
  }
}