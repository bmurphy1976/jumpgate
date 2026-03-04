// Shared theme switcher — works with both <style> (dashboard) and <link> (admin).
(() => {
	const themeSelect = document.getElementById('theme-select');
	if (!themeSelect) return;

	themeSelect.addEventListener('change', () => {
		const themeEl = document.getElementById('theme-stylesheet');
		if (!themeEl) return;

		const isInline = themeEl.tagName === 'STYLE';
		const theme = themeSelect.value || '';
		const versionSuffix = window.THEME_HASHES && window.THEME_HASHES[theme]
			? `?v=${window.THEME_HASHES[theme]}` : '';
		const nextHref = `/static/themes/${encodeURIComponent(theme)}.css${versionSuffix}`;

		if (isInline) {
			fetch(nextHref)
				.then((r) => r.ok ? r.text() : Promise.reject())
				.then((css) => { themeEl.textContent = css; })
				.catch(() => {});
		} else {
			try {
				const oldLink = themeEl;
				const nextLink = document.createElement('link');
				nextLink.rel = 'stylesheet';
				nextLink.id = 'theme-stylesheet';
				nextLink.href = nextHref;
				let done = false;
				const finish = (ok) => {
					if (done) return;
					done = true;
					if (ok) {
						if (oldLink.parentNode) oldLink.parentNode.removeChild(oldLink);
					} else {
						if (nextLink.parentNode) nextLink.parentNode.removeChild(nextLink);
					}
				};
				nextLink.addEventListener('load', () => { finish(true); });
				nextLink.addEventListener('error', () => { finish(false); });
				oldLink.parentNode.insertBefore(nextLink, oldLink.nextSibling);
				setTimeout(() => { if (!done) finish(!!nextLink.sheet); }, 500);
			} catch (e) {
				themeEl.setAttribute('href', nextHref);
			}
		}

		fetch('/set-theme', {
			method: 'POST',
			credentials: 'same-origin',
			headers: {'Content-Type': 'application/x-www-form-urlencoded'},
			body: `theme=${encodeURIComponent(theme)}`
		});
	});
})();
