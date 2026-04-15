// __root is the element's wrapper div: #tf-element-stagger-title
// All DOM access should remain within __root.

// Read a --tf-ui-* duration variable and return it in seconds (GSAP uses seconds).
function _st_getDuration(varName, fallbackMs) {
  var str = getComputedStyle(__root).getPropertyValue(varName).trim();
  var ms = str.endsWith('ms') ? parseFloat(str) : parseFloat(str) * 1000;
  return (isNaN(ms) || ms <= 0 ? fallbackMs : ms) / 1000;
}

// lettering.js rewrites a span's textContent into individual char spans and stores
// the original text in aria-label. Before re-lettering (e.g. after a bind update),
// restore each line to plain text so lettering() doesn't double-wrap.
function _st_reletter() {
  $(__root).find('.stagger-title__line').each(function() {
    var label = this.getAttribute('aria-label');
    if (label !== null) {
      this.textContent = label;
      this.removeAttribute('aria-label');
    }
  });
  $(__root).find('.stagger-title__line').lettering();
}

document.addEventListener('tf:fire', function(e) {
  if (e.detail.element !== 'stagger-title') return;

  if (e.detail.event === 'show') {
    // Re-letter on every show so any dock text changes are picked up.
    _st_reletter();
    var dur = _st_getDuration('--tf-ui-show-duration', 500);
    TweenMax.staggerFromTo(
      $(__root).find('.stagger-title__line span').toArray(),
      dur,
      { ease: Back.easeOut.config(1.7), opacity: 0, bottom: -80 },
      { ease: Back.easeOut.config(1.7), opacity: 1, bottom: 0 },
      0.05
    );
  }

  if (e.detail.event === 'hide') {
    e.preventDefault(); // suppress runtime default — GSAP needs custom timing
    var dur = _st_getDuration('--tf-ui-hide-duration', 500);
    var chars = $(__root).find('.stagger-title__line span').toArray();
    // Total animation time: last char starts at stagger * (n-1), then runs for dur seconds.
    var totalMs = (dur + 0.05 * Math.max(chars.length - 1, 0)) * 1000 + 50;

    TweenMax.staggerFromTo(
      chars,
      dur,
      { ease: Back.easeIn.config(1.7), opacity: 1, bottom: 0 },
      { ease: Back.easeIn.config(1.7), opacity: 0, bottom: -80 },
      0.05
    );

    setTimeout(function() {
      // Restore plain text so bind updates work cleanly before the next show.
      $(__root).find('.stagger-title__line').each(function() {
        var label = this.getAttribute('aria-label');
        if (label !== null) {
          this.textContent = label;
          this.removeAttribute('aria-label');
        }
      });
      __root.classList.remove('tf-fired');
    }, totalMs);
  }
});
