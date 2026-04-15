document.addEventListener('tf:fire', function(e) {
  if (e.detail.element !== 'david-simon') return;

  var event = e.detail.event;

  if (event === 'hide') {
    var durStr = getComputedStyle(__root).getPropertyValue('--tf-ui-hide-duration').trim();
    var ms = durStr.endsWith('ms') ? parseFloat(durStr) : parseFloat(durStr) * 1000;
    if (!ms || isNaN(ms)) ms = 700;

    __root.classList.add('tf-hiding');
    setTimeout(function() {
      __root.classList.remove('tf-fired', 'tf-hiding');
    }, ms);
  }

  if (event === 'flash') {
    __root.classList.remove('tf-flashing');
    void __root.offsetWidth;
    __root.classList.add('tf-flashing');
  }

  if (event === 'pulse') {
    __root.classList.remove('tf-pulsing');
    void __root.offsetWidth;
    __root.classList.add('tf-pulsing');
  }
});

document.addEventListener('tf:fire', function(e) {
  if (e.detail.element !== 'david-simon') return;
  if (e.detail.event !== 'show') return;

  __root.classList.remove('tf-pulsing');
});
