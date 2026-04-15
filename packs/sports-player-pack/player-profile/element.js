document.addEventListener('tf:fire', function(e) {
  if (e.detail.element !== 'player-profile') return;
  if (e.detail.event !== 'hide') return;

  var durStr = getComputedStyle(__root).getPropertyValue('--tf-ui-hide-duration').trim();
  var ms = durStr.endsWith('ms') ? parseFloat(durStr) : parseFloat(durStr) * 1000;
  if (!ms || isNaN(ms)) ms = 600;

  __root.classList.add('tf-hiding');
  setTimeout(function() {
    __root.classList.remove('tf-fired', 'tf-hiding');
  }, ms);
});
