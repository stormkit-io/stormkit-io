import { useMemo } from 'react'
import Box from '@mui/material/Box'

type PathRef = React.RefObject<SVGPathElement>

interface Props {
  pathRef: PathRef | PathRef[]
  strokeWidth?: number
  stroke?: string
  animation?: boolean
}

export default function DrawContainer({
  pathRef,
  animation,
  stroke,
  strokeWidth = 12.6,
}: Props) {
  const refs = useMemo(() => {
    return Array.isArray(pathRef) ? pathRef : [pathRef]
  }, [pathRef])

  return (
    <Box
      component="svg"
      id="tech-sk-deploy-connector"
      xmlns="http://www.w3.org/2000/svg"
      width="100%"
      height="100%"
      className={animation ? 'animated-svg-path' : ''}
      strokeDasharray="250"
      strokeDashoffset="1000"
      sx={{
        position: 'absolute',
        left: 0,
        top: 0,
        display: { xs: 'none', md: 'block' },
      }}
    >
      {refs.map((pathRef, index) => (
        <path
          key={index}
          ref={pathRef}
          opacity="0.5"
          fill="none"
          stroke={stroke}
          strokeWidth={strokeWidth}
        />
      ))}
    </Box>
  )
}
