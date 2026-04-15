(function() {
  var autoHideTimer = null;

  document.addEventListener('tf:fire', function(e) {
    if (e.detail.element !== 'nameplate') return;

    if (e.detail.event === 'show') {
      if (autoHideTimer) clearTimeout(autoHideTimer);
      autoHideTimer = setTimeout(function() {
        if (!__root.classList.contains('tf-fired')) return;
        triggerHide();
      }, 4000);
    }

    if (e.detail.event === 'hide') {
      e.preventDefault();
      if (autoHideTimer) { clearTimeout(autoHideTimer); autoHideTimer = null; }
      triggerHide();
    }

    function triggerHide() {
      __root.classList.add('tf-hiding');
      var dur = parseInt(getComputedStyle(__root).getPropertyValue('--tf-ui-hide-duration')) || 400;
      setTimeout(function() {
        __root.classList.remove('tf-fired', 'tf-hiding');
      }, dur);
    }
  });
})();
