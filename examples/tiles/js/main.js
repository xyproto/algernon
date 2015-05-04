// From: http://masonry.desandro.com/faq.html

var transitionProp = getStyleProperty('transition');
var transitionEndEvent = {
  WebkitTransition: 'webkitTransitionEnd',
  MozTransition: 'transitionend',
  OTransition: 'otransitionend',
  transition: 'transitionend'
}[transitionProp];

docReady(function() {

  var container = document.querySelector('.masonry');
  var msnry = new Masonry(container, {
    itemSelector: '.item',
    columnWidth: '.grid-sizer'
  });

  eventie.bind(container, 'click', function(event) {
    // don't proceed if item content was not clicked on
    var target = event.target;
    if (!classie.has(target, 'item-content')) {
      return;
    }

    var previousContentSize = getSize(target);
    // disable transition
    target.style[transitionProp] = 'none';
    // set current size
    target.style.width = previousContentSize.width + 'px';
    target.style.height = previousContentSize.height + 'px';

    var itemElem = target.parentNode;
    classie.toggleClass(itemElem, 'is-expanded');

    // force redraw
    var redraw = target.offsetWidth;
    // renable default transition
    target.style[transitionProp] = '';

    // reset 100%/100% sizing after transition end
    if (transitionProp) {
      var onTransitionEnd = function() {
        target.style.width = '';
        target.style.height = '';
        target.removeEventListener(transitionEndEvent, onTransitionEnd, false);
      };
      target.addEventListener(transitionEndEvent, onTransitionEnd, false);
    }

    // set new size
    var size = getSize(itemElem);
    target.style.width = size.width + 'px';
    target.style.height = size.height + 'px';
    redraw = null; // for JSHint

    msnry.layout();
  });

});
