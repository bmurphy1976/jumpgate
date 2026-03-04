(() => {
    // Search
    const searchInput = document.getElementById('search-input');
    const searchIcon = document.querySelector('.SearchBar_Icon');
    if (searchInput && searchIcon) {
        // Enable search when JS loads
        searchInput.disabled = false;
        searchIcon.style.display = 'block';

        // Auto-focus search bar on desktop only (not on mobile/touch devices)
        if (!window.matchMedia('(pointer: coarse)').matches) {
            searchInput.focus();
        }

        // Get all searchable links
        const favoriteLinks = document.querySelectorAll('.AppCard');
        const bookmarkCards = document.querySelectorAll('.BookmarkCard');
        const bookmarkLinks = document.querySelectorAll('.BookmarkCard_Bookmarks a');
        const allLinks = [...favoriteLinks, ...bookmarkLinks];

        const filterLinks = () => {
            const query = searchInput.value.trim().toLowerCase();
            const visibleLinks = [];

            allLinks.forEach((link) => {
                const text = link.textContent.toLowerCase();
                const keywords = (link.dataset.keywords || '').toLowerCase();
                const matches = text.includes(query) || keywords.includes(query);

                if (matches) {
                    link.style.display = '';
                    visibleLinks.push(link);
                } else {
                    link.style.display = 'none';
                }
            });

            // Hide empty bookmark categories
            bookmarkCards.forEach((card) => {
                const linksInCategory = card.querySelectorAll('.BookmarkCard_Bookmarks a');
                const hasVisibleLinks = [...linksInCategory].some((link) => link.style.display !== 'none');
                card.style.display = hasVisibleLinks ? '' : 'none';
            });

            return visibleLinks;
        };

        // Filter on input
        searchInput.addEventListener('input', filterLinks);

        // Handle keyboard events on search input
        searchInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                const query = searchInput.value.trim();
                if (query === '') return;

                const visibleLinks = filterLinks();
                if (visibleLinks.length > 0) {
                    window.location.href = visibleLinks[0].href;
                }
            } else if (e.key === 'Escape' || (e.ctrlKey && e.key === 'u')) {
                e.preventDefault();
                searchInput.value = '';
                filterLinks();
            }
        });

        // Global escape handler - focus search bar and scroll to top when escape pressed anywhere
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && document.activeElement !== searchInput) {
                e.preventDefault();
                window.scrollTo(0, 0);
                searchInput.focus();
            }
        });
    }

    // Weather location cookies (progressive enhancement)
    (() => {
        let isWeatherVisible = true;
        if (window.matchMedia && !window.matchMedia('(min-width:600px)').matches) {
            isWeatherVisible = false;
        }
        if (!document.querySelector('.WeatherWidget')) {
            isWeatherVisible = false;
        }
        if (!isWeatherVisible) return;

        const hasLocationCookies = document.cookie.includes('weather_lat') && document.cookie.includes('weather_lon');

        if (!hasLocationCookies && navigator && 'geolocation' in navigator) {
            navigator.geolocation.getCurrentPosition(
                (position) => {
                    const lat = position.coords.latitude;
                    const lon = position.coords.longitude;

                    const expires = new Date();
                    expires.setDate(expires.getDate() + 30);
                    document.cookie = `weather_lat=${lat}; expires=${expires.toUTCString()}; path=/; SameSite=Lax`;
                    document.cookie = `weather_lon=${lon}; expires=${expires.toUTCString()}; path=/; SameSite=Lax`;

                    window.location.reload();
                },
                () => {
                    // denied/failed: fall back to server default location
                }
            );
        }
    })();
})();
