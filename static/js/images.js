window.onload = function() {
	// submit with enter key
	searchbar.onkeydown = function(e) {
		if (e.keyCode == 13){
			location.href = './images?t=' + e.target.value;
		}
	};
}