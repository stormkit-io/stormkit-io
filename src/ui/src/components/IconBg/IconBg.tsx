import Box, { BoxProps } from "@mui/material/Box";

interface Props extends BoxProps {
  children: React.ReactNode;
}

export default function IconBg({ children, sx }: Props) {
  return (
    <Box
      component="span"
      sx={{
        bgcolor: "rgba(255,255,255,0.1)",
        mr: 1,
        borderRadius: "50%",
        width: 26,
        height: 26,
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        ...sx,
      }}
    >
      {children}
    </Box>
  );
}
