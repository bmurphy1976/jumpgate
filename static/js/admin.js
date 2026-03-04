// HTMX save feedback
function saveFeedbackTarget(elt) {
	if (!elt) return null;
	// Hidden inputs (keywords): chips handle their own feedback
	if (elt.type === 'hidden') return null;
	const swap = elt.getAttribute('hx-swap') || '';
	const hxTarget = elt.getAttribute('hx-target');
	// outerHTML swap targeting an ancestor (sort): show on the category card
	if (swap.startsWith('outerHTML') && hxTarget && hxTarget !== 'this'
		&& !hxTarget.startsWith('closest ')) {
		return elt.closest('.category-card') || elt;
	}
	// closest-targeting (delete bookmark): resolve it
	if (hxTarget?.startsWith('closest ')) return elt.closest(hxTarget.slice(8)) || elt;
	// Everything else (self-targeting toggles, inputs with swap:none): self
	return elt;
}

document.addEventListener('htmx:beforeRequest', (e) => {
	const target = saveFeedbackTarget(e.detail.elt);
	if (!target) return;
	target.classList.add('save-pending');
	// Store metadata for reconnection after outerHTML swaps
	if (target.dataset?.categoryId) e.detail.elt._fbCatId = target.dataset.categoryId;
	if (target.id) e.detail.elt._fbId = target.id;
});

// HTMX error feedback — persists until a successful request clears it
function markError(e) {
	if (e.detail.elt?.dataset.retrying !== undefined) return;
	const target = saveFeedbackTarget(e.detail.elt);
	if (target) target.classList.add('save-err');
}
document.addEventListener('htmx:responseError', markError);
document.addEventListener('htmx:sendError', markError);

// Save feedback removal + retry with backoff (2 attempts, then error banner)
const retryMap = new Map();

document.addEventListener('htmx:afterRequest', (e) => {
	const elt = e.detail.elt;
	if (!elt) return;

	let target = saveFeedbackTarget(elt);
	// Reconnect: outerHTML swap detaches the old element, find the new one
	if (!target?.isConnected) {
		if (elt._fbId) target = document.getElementById(elt._fbId);
		else if (elt._fbCatId) target = document.querySelector(`[data-category-id="${elt._fbCatId}"]`);
	}
	delete elt._fbCatId;
	delete elt._fbId;

	if (target?.isConnected) {
		target.classList.remove('save-pending');
	}

	if (e.detail.successful) {
		retryMap.delete(elt);
		delete elt.dataset.retrying;
		if (target?.isConnected) target.classList.remove('save-err');
		return;
	}

	const count = (retryMap.get(elt) || 0) + 1;

	if (count > 2) {
		retryMap.delete(elt);
		delete elt.dataset.retrying;
		showErrorBanner();
		return;
	}

	retryMap.set(elt, count);
	elt.dataset.retrying = '';
	if (target?.isConnected) target.classList.add('save-pending');

	const rc = e.detail.requestConfig;
	setTimeout(() => {
		// Declarative HTMX elements: re-trigger natively to preserve swap/target
		if (elt.hasAttribute('hx-get') || elt.hasAttribute('hx-post') ||
			elt.hasAttribute('hx-put') || elt.hasAttribute('hx-delete')) {
			const trigger = (elt.getAttribute('hx-trigger') || '').split(/[\s,]/)[0]
				|| (elt.matches('input,select,textarea') ? 'change' : 'click');
			htmx.trigger(elt, trigger);
		} else {
			// Programmatic sources (reorder, move): replay via htmx.ajax
			htmx.ajax(rc.verb, rc.path, { source: elt, values: rc.parameters, swap: 'none' });
		}
	}, 1000 * count);
});

function showErrorBanner() {
	if (document.getElementById('errorBanner')) return;
	const banner = document.createElement('div');
	banner.id = 'errorBanner';
	banner.className = 'error-banner';

	const msg = document.createElement('span');
	msg.textContent = 'A save operation failed. Reload to ensure your changes are saved.';

	const reload = document.createElement('button');
	reload.className = 'btn btn-sm';
	reload.textContent = 'Reload';
	reload.onclick = () => window.location.reload();

	const dismiss = document.createElement('button');
	dismiss.className = 'btn-icon';
	dismiss.textContent = '\u00d7';
	dismiss.setAttribute('aria-label', 'Dismiss');
	dismiss.onclick = () => banner.remove();

	banner.append(msg, reload, dismiss);
	document.body.prepend(banner);
}

// Edit row accordion
function toggleEditRow(row) {
	const editRow = row.nextElementSibling;
	if (!editRow || !editRow.classList.contains('edit-row')) return;
	const isOpening = !editRow.classList.contains('active');
	document.querySelectorAll('.edit-row.active').forEach((r) => {
		r.classList.remove('active');
	});
	if (isOpening) {
		editRow.classList.add('active');
	}
}

// Settings collapse
function toggleSettings() {
	document.querySelector('.settings-header').classList.toggle('open');
	document.getElementById('settingsBody').classList.toggle('open');
}

// Confirm dialog
let pendingDeleteEl = null;
function showConfirm(msg, triggerEl) {
	document.getElementById('confirmMessage').textContent = msg;
	document.getElementById('confirmOverlay').classList.add('active');
	pendingDeleteEl = triggerEl;
}
function hideConfirm() {
	document.getElementById('confirmOverlay').classList.remove('active');
	pendingDeleteEl = null;
}
document.getElementById('confirmOverlay').addEventListener('click', hideConfirm);
document.getElementById('confirmDeleteBtn').addEventListener('click', () => {
	if (pendingDeleteEl) {
		htmx.trigger(pendingDeleteEl, 'confirmed');
	}
	hideConfirm();
});

// Auto-expand newly added bookmark, auto-focus newly added category
document.addEventListener('htmx:afterSwap', (e) => {
	const elt = e.detail.elt;
	if (!elt || !elt.classList) return;
	if (elt.classList.contains('bookmark-table')) {
		const entry = elt.querySelector('.bookmark-entry:first-child');
		if (entry) {
			const editRow = entry.querySelector('.edit-row');
			if (editRow) {
				document.querySelectorAll('.edit-row.active').forEach((r) => { r.classList.remove('active'); });
				editRow.classList.add('active');
				const nameInput = editRow.querySelector('input[name="name"]');
				if (nameInput) nameInput.focus();
			}
		}
	}
	if (elt.id === 'bookmarkCategories') {
		const card = elt.querySelector('.category-card:first-child');
		if (card) {
			const nameInput = card.querySelector('.category-name-input');
			if (nameInput) { nameInput.focus(); nameInput.select(); }
		}
	}
});

// Icon picker
let activePickerBtn = null;
document.addEventListener('click', (e) => {
	const btn = e.target.closest('.icon-picker-btn');
	if (!btn) return;
	activePickerBtn = btn;
	const rect = btn.getBoundingClientRect();
	const picker = document.getElementById('iconPicker');
	picker.style.top = `${rect.bottom + 4}px`;
	picker.style.left = `${Math.min(rect.left, window.innerWidth - 330)}px`;
});
document.getElementById('iconPickerOverlay').addEventListener('toggle', (e) => {
	if (e.newState === 'open') {
		const s = document.getElementById('iconSearchInput');
		s.value = '';
		s.focus();
		htmx.trigger(s, 'input');
	} else {
		const btn = activePickerBtn;
		activePickerBtn = null;
		if (btn) {
			btn.focus();
			btn.classList.add('focus-ring');
			btn.addEventListener('blur', () => { btn.classList.remove('focus-ring'); }, { once: true });
		}
	}
});

// Category overflow menu positioning and cleanup
let activeOverflowMenu = null;
let activeOverflowBtn = null;

function positionOverflowMenu(menu, btn) {
	if (!btn?.isConnected) return;
	requestAnimationFrame(() => {
		const rect = btn.getBoundingClientRect();
		menu.style.position = 'fixed';
		menu.style.top = `${rect.bottom + 4}px`;
		menu.style.left = 'auto';
		menu.style.right = `${window.innerWidth - rect.right}px`;
	});
}

document.addEventListener('toggle', (e) => {
	if (!e.target.classList.contains('category-overflow-menu')) return;

	if (e.newState === 'open') {
		const menu = e.target;

		// Clear save feedback from the container itself
		menu.classList.remove('save-pending', 'save-err');

		// Clear any stale save feedback from children
		menu.querySelectorAll('.save-pending, .save-err').forEach(el => {
			el.classList.remove('save-pending', 'save-err');
		});

		const btn = document.querySelector(`[popovertarget="${menu.id}"]`);
		if (!btn) return;

		activeOverflowMenu = menu;
		activeOverflowBtn = btn;
		positionOverflowMenu(menu, btn);
	} else if (e.newState === 'closed') {
		activeOverflowMenu = null;
		activeOverflowBtn = null;

		// Clear save feedback from container and children when popover closes
		e.target.classList.remove('save-pending', 'save-err');
		e.target.querySelectorAll('.save-pending, .save-err').forEach(el => {
			el.classList.remove('save-pending', 'save-err');
		});
	}
}, true);

// Update overflow menu position on scroll
document.addEventListener('scroll', () => {
	if (activeOverflowMenu && activeOverflowBtn) {
		positionOverflowMenu(activeOverflowMenu, activeOverflowBtn);
	}
}, true);

const searchInput = document.getElementById('searchInput');

document.addEventListener('keydown', (e) => {
	if (e.key === 'Escape') {
		const confirmActive = document.getElementById('confirmOverlay').classList.contains('active');
		hideConfirm();
		if (confirmActive) return;
		if (document.getElementById('iconPickerOverlay').matches(':popover-open')) return;

		const active = document.activeElement;

		if (active.classList.contains('icon-picker-btn')) {
			const iconInput = active.closest('.icon-input-group').querySelector('input');
			if (iconInput) iconInput.focus();
			e.preventDefault();
			return;
		}

		const editRow = active.closest('.edit-row');
		if (editRow && editRow.classList.contains('active')) {
			const bookmarkRow = editRow.previousElementSibling;
			if (bookmarkRow) bookmarkRow.focus({ focusVisible: true });
			e.preventDefault();
			return;
		}

		const bookmarkRowEl = active.closest('.bookmark-row');
		if (bookmarkRowEl) {
			if (active !== bookmarkRowEl) {
				bookmarkRowEl.focus({ focusVisible: true });
				e.preventDefault();
				return;
			}
			const nextEditRow = bookmarkRowEl.nextElementSibling;
			if (nextEditRow && nextEditRow.classList.contains('edit-row') && nextEditRow.classList.contains('active')) {
				nextEditRow.classList.remove('active');
			} else {
				e.preventDefault();
				window.scrollTo(0, 0);
				searchInput.focus();
			}
			return;
		}

		if (active !== searchInput) {
			e.preventDefault();
			window.scrollTo(0, 0);
			searchInput.focus();
		}
	}
});

function selectIcon(icon) {
	if (!activePickerBtn) return;
	const inputGroup = activePickerBtn.closest('.icon-input-group');
	if (inputGroup) {
		const input = inputGroup.querySelector('input');
		input.value = icon;
		htmx.trigger(input, 'change');
	}
	const editRow = activePickerBtn.closest('.edit-row');
	if (editRow) {
		const bookmarkRow = editRow.previousElementSibling;
		if (bookmarkRow) {
			const tdIcon = bookmarkRow.querySelector('.td-icon .mdi');
			if (tdIcon) tdIcon.className = `mdi mdi-${icon}`;
		}
	}
	document.getElementById('iconPickerOverlay').hidePopover();
}

// Keyword tag input
function commitKeyword(input) {
	const raw = input.value.replace(/[^\p{L}\p{N}]/gu, '').trim();
	if (raw) {
		const container = input.closest('.keyword-tags');
		const hidden = container.closest('.keywords-group').querySelector('input[type="hidden"]');
		const current = hidden.value ? hidden.value.split(' ') : [];
		if (!current.includes(raw)) {
			current.push(raw);
			const chip = document.createElement('span');
			chip.className = 'keyword-tag';
			chip.dataset.keyword = raw;
			chip.innerHTML = `${raw}<button type="button" class="keyword-tag-remove" onclick="removeKeyword(this)" aria-label="Remove ${raw}">\u00d7</button>`;
			container.insertBefore(chip, input);
			input.removeAttribute('placeholder');
			hidden.value = current.join(' ');
			chip.classList.add('save-pending');
			htmx.trigger(hidden, 'change');
		}
	}
	input.value = '';
}
function removeKeyword(btn) {
	const chip = btn.parentElement;
	const container = chip.parentElement;
	const hidden = container.closest('.keywords-group').querySelector('input[type="hidden"]');
	const kw = chip.dataset.keyword;
	// Don't remove chip yet — keep it visible until server confirms
	const current = hidden.value ? hidden.value.split(' ').filter((k) => k !== kw) : [];
	hidden.value = current.join(' ');
	chip.classList.add('save-pending');
	if (!hidden._pendingRemovals) hidden._pendingRemovals = new Map();
	hidden._pendingRemovals.set(kw, { chip, container });
	htmx.trigger(hidden, 'change');
}
function handleKeywordKeydown(e) {
	if (e.key === 'Enter' || e.key === ' ') {
		e.preventDefault();
		commitKeyword(e.target);
	} else if (e.key === 'Backspace' && e.target.value === '') {
		const container = e.target.closest('.keyword-tags');
		const chips = container.querySelectorAll('.keyword-tag');
		if (chips.length) removeKeyword(chips[chips.length - 1].querySelector('.keyword-tag-remove'));
	}
}

// Keyword save cleanup — each chip owns its own feedback
document.addEventListener('htmx:afterRequest', (e) => {
	const elt = e.detail.elt;
	if (elt?.type !== 'hidden') return;

	// Clear save-pending from all chips in this keyword group
	const group = elt.closest('.keywords-group');
	if (!group) return;
	group.querySelectorAll('.keyword-tag.save-pending').forEach(c => c.classList.remove('save-pending'));

	// Process pending removals
	const removals = elt._pendingRemovals;
	if (!removals?.size) return;

	if (e.detail.successful) {
		for (const [, { chip, container }] of removals) {
			chip.remove();
			if (!container.querySelector('.keyword-tag')) {
				container.querySelector('.keyword-tag-input')?.setAttribute('placeholder', 'Add keywords\u2026');
			}
		}
	} else {
		// Restore all pending keywords on failure
		const current = elt.value ? elt.value.split(' ') : [];
		for (const [keyword] of removals) {
			if (!current.includes(keyword)) current.push(keyword);
		}
		elt.value = current.join(' ');
	}
	elt._pendingRemovals = null;
});

// Search
function filterBookmarks() {
	const q = searchInput.value.trim().toLowerCase();
	const cards = document.querySelectorAll('.category-card');
	cards.forEach((card) => {
		const catInput = card.querySelector('.category-name-input');
		const catName = catInput ? catInput.value.toLowerCase() : '';
		const rows = card.querySelectorAll('.bookmark-table tr.bookmark-row');
		let anyVisible = false;
		rows.forEach((row) => {
			const name = row.querySelector('.td-name');
			const url = row.querySelector('.td-url');
			const text = `${name ? name.textContent.toLowerCase() : ''} ${catName} ${url ? url.textContent.toLowerCase() : ''} ${(row.dataset.keywords || '').toLowerCase()}`;
			const editRow = row.nextElementSibling;
			if (q === '' || text.includes(q)) {
				row.style.display = '';
				if (editRow && editRow.classList.contains('edit-row')) editRow.style.display = '';
				anyVisible = true;
			} else {
				row.style.display = 'none';
				if (editRow && editRow.classList.contains('edit-row')) editRow.style.display = 'none';
			}
		});
		card.style.display = (q === '' || anyVisible) ? '' : 'none';
	});
	const settings = document.querySelector('.settings-card');
	if (settings) {
		settings.style.display = (q === '' || 'settings'.includes(q)) ? '' : 'none';
	}
}
searchInput.addEventListener('input', filterBookmarks);
searchInput.addEventListener('keydown', (e) => {
	if (e.key === 'Escape' || (e.ctrlKey && e.key === 'u')) {
		e.preventDefault();
		searchInput.value = '';
		filterBookmarks();
	}
});
if (!window.matchMedia('(pointer: coarse)').matches) {
	searchInput.focus();
}

// SortableJS for category reordering (non-favorites only)
const bookmarkCategories = document.getElementById('bookmarkCategories');
if (bookmarkCategories) {
	Sortable.create(bookmarkCategories, {
		handle: '.drag-handle',
		animation: 150,
		onEnd: (evt) => {
			const ids = [...bookmarkCategories.querySelectorAll('.category-card')]
				.map((el) => parseInt(el.dataset.categoryId));
			htmx.ajax('POST', '/admin/categories/reorder', {
				source: evt.item,
				values: { order: JSON.stringify(ids) },
				swap: 'none'
			});
			// Reinit bookmark sortables — SortableJS group registry can break when parent elements are moved
			bookmarkCategories.querySelectorAll('.bookmark-table').forEach((table) => {
				const s = Sortable.get(table);
				if (s) s.destroy();
				initBookmarkSortable(table);
			});
		}
	});
}

// SortableJS for bookmark reordering within and between categories
let sortableDragItem = null;
let headerDropped = false;

function initBookmarkSortable(table) {
	Sortable.create(table, {
		handle: '.drag-handle',
		draggable: 'tbody.bookmark-entry',
		group: 'bookmarks',
		animation: 150,
		onStart: (evt) => { sortableDragItem = evt.item; },
		onEnd: (evt) => {
			if (headerDropped) {
				headerDropped = false;
				sortableDragItem = null;
				return;
			}
			sortableDragItem = null;
			const bookmarkId = parseInt(evt.item.dataset.bookmarkId);
			const newCategoryId = evt.to.closest('.category-card').dataset.categoryId;
			const oldCategoryId = evt.from.closest('.category-card').dataset.categoryId;
			const ids = [...evt.to.querySelectorAll('.bookmark-entry')]
				.map((el) => parseInt(el.dataset.bookmarkId));

			if (oldCategoryId !== newCategoryId) {
				htmx.ajax('POST', `/admin/bookmarks/${bookmarkId}/move`, {
					source: evt.to.closest('.category-card'),
					values: {
						target_category_id: newCategoryId,
						order: JSON.stringify(ids)
					},
					swap: 'none'
				});
			} else {
				htmx.ajax('POST', '/admin/bookmarks/reorder', {
					source: evt.to.closest('.category-card'),
					values: {
						category_id: newCategoryId,
						order: JSON.stringify(ids)
					},
					swap: 'none'
				});
			}
		}
	});
}

document.querySelectorAll('.bookmark-table').forEach(initBookmarkSortable);

// Drop bookmarks on category headers (delegated so new/moved categories work automatically)
if (bookmarkCategories) {
	bookmarkCategories.addEventListener('dragover', (e) => {
		if (!sortableDragItem) return;
		const header = e.target.closest('.category-header');
		if (!header) return;
		e.preventDefault();
		header.classList.add('drag-over');
	});
	bookmarkCategories.addEventListener('dragleave', (e) => {
		const header = e.target.closest('.category-header');
		if (header && !header.contains(e.relatedTarget)) {
			header.classList.remove('drag-over');
		}
	});
	bookmarkCategories.addEventListener('drop', (e) => {
		const header = e.target.closest('.category-header');
		if (!header || !sortableDragItem) return;
		e.preventDefault();
		header.classList.remove('drag-over');
		const item = sortableDragItem;
		const bookmarkId = parseInt(item.dataset.bookmarkId);
		const card = header.closest('.category-card');
		const table = card.querySelector('.bookmark-table');
		const newCategoryId = card.dataset.categoryId;
		headerDropped = true;
		setTimeout(() => {
			table.appendChild(item);
			const ids = [...table.querySelectorAll('.bookmark-entry')]
				.map((el) => parseInt(el.dataset.bookmarkId));
			htmx.ajax('POST', `/admin/bookmarks/${bookmarkId}/move`, {
				source: card,
				values: {
					target_category_id: newCategoryId,
					order: JSON.stringify(ids)
				},
				swap: 'none'
			});
		}, 0);
	});
}

// Reinitialize SortableJS on any new bookmark tables after swaps
document.addEventListener('htmx:afterSwap', () => {
	document.querySelectorAll('.bookmark-table').forEach((table) => {
		if (!Sortable.get(table)) {
			initBookmarkSortable(table);
		}
	});
});
