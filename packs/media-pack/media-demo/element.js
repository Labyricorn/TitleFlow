// __root is the element's wrapper div: #tf-element-media-demo

// Get audio element for playback
var _mediaAudioElement = null;

// Play sound using HTML5 audio element
function _media_playAudio() {
  if (!_mediaAudioElement) {
    _mediaAudioElement = document.getElementById('media-demo-audio');
  }

  if (!_mediaAudioElement) return;

  // Reset playback to start
  _mediaAudioElement.currentTime = 0;
  _mediaAudioElement.play().catch(err => {
    console.error('Failed to play audio:', err);
  });

  // Trigger pulse feedback
  _media_triggerPulse();
}

// Trigger visual pulse animation when sound plays
function _media_triggerPulse() {
  var feedback = __root.querySelector('.media-demo__sound-feedback');
  if (!feedback) return;

  // Find center of container for pulse origin
  var container = __root.querySelector('.media-demo__container');
  var rect = container.getBoundingClientRect();
  var centerX = rect.left + rect.width / 2;
  var centerY = rect.top + rect.height / 2;

  // Position feedback element
  feedback.style.left = centerX + 'px';
  feedback.style.top = centerY + 'px';

  // Remove previous animation
  feedback.style.animation = 'none';

  // Trigger reflow to restart animation
  void feedback.offsetWidth;

  // Apply pulse animation
  feedback.style.animation = 'media-pulse 600ms ease-out forwards';
}

// Fire event listener
document.addEventListener('tf:fire', function(e) {
  if (e.detail.element !== 'media-demo') return;

  if (e.detail.event === 'show') {
    // .tf-fired class is already applied by runtime, CSS animation handles entrance
  }

  if (e.detail.event === 'hide') {
    e.preventDefault();
    // Apply hide animation: add .tf-hiding class and wait for CSS animation
    __root.classList.add('tf-hiding');
    var duration = getComputedStyle(__root).getPropertyValue('--tf-ui-hide-duration');
    var ms = duration.endsWith('ms') ? parseFloat(duration) : parseFloat(duration) * 1000;

    setTimeout(function() {
      __root.classList.remove('tf-fired', 'tf-hiding');
    }, ms);
  }

  if (e.detail.event === 'play-sound') {
    _media_playAudio();
  }
});
