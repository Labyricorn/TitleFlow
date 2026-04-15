(function() {
  document.addEventListener('tf:fire', function(e) {
    if (e.detail.element !== 'ticker') return;

    if (e.detail.event === 'show') {
      var scroll = __root.querySelector('.tk-scroll');
      if (scroll) {
        scroll.style.animationPlayState = 'running';
      }
    }

    if (e.detail.event === 'hide') {
      e.preventDefault();
      triggerHide();
    }

    function triggerHide() {
      __root.classList.add('tf-hiding');
      var dur = parseInt(getComputedStyle(__root).getPropertyValue('--tf-ui-hide-duration')) || 400;
      setTimeout(function() {
        __root.classList.remove('tf-fired', 'tf-hiding');
        var scroll = __root.querySelector('.tk-scroll');
        if (scroll) {
          scroll.style.animationPlayState = 'paused';
        }
      }, dur);
    }
  });
})();
