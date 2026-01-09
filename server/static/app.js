// Prevent showing cached protected pages after logout.
window.addEventListener("pageshow", function (event) {
	if (event.persisted) {
		window.location.replace("/");
	}
});
