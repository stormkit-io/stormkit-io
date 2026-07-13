type Position = 'right-middle' | 'bottom-middle' | 'left-middle' | 'top-middle'

interface Options {
  from?: Position
  to?: Position
  distortX?: number
  distortY?: number
}

function calculatePos(divA: HTMLElement, position: Position) {
  switch (position) {
    case 'bottom-middle': {
      return {
        x: divA.offsetLeft + divA.offsetWidth / 2,
        y: divA.offsetTop + divA.offsetHeight,
      }
    }

    case 'top-middle': {
      return {
        x: divA.offsetLeft + divA.offsetWidth / 2,
        y: divA.offsetTop,
      }
    }

    case 'left-middle': {
      return {
        x: divA.offsetLeft,
        y: divA.offsetTop + divA.offsetHeight / 2,
      }
    }

    case 'right-middle': {
      return {
        x: divA.offsetLeft + divA.offsetWidth,
        y: divA.offsetTop + divA.offsetHeight / 2,
      }
    }

    default: {
      return {
        x: 0,
        y: 0,
      }
    }
  }
}

export function drawConnector(
  connector: SVGPathElement | null,
  divA: HTMLElement | null,
  divB: HTMLElement | null,
  options: Options = {}
) {
  if (!divA || !divB || !connector) {
    return
  }

  const posnA = calculatePos(divA, options.from || 'right-middle')
  const posnB = calculatePos(divB, options.to || 'left-middle')
  const { distortX = 0, distortY = 0 } = options

  const dStr =
    'M' +
    posnA.x +
    ',' +
    posnA.y +
    ' ' +
    'C' +
    (posnA.x + distortX) +
    ',' +
    (posnA.y + distortY) +
    ' ' +
    (posnB.x - distortX) +
    ',' +
    (posnB.y - distortY) +
    ' ' +
    posnB.x +
    ',' +
    posnB.y

  connector.setAttribute('d', dStr)
}
