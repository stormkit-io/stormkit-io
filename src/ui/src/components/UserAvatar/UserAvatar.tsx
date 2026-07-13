import AccountCircleIcon from "@mui/icons-material/AccountCircle";
import Box, { BoxProps } from "@mui/material/Box";

interface Props extends BoxProps {
  user: User;
}

export default function UserAvatar({ user, sx }: Props) {
  const width = 24;
  const height = 24;

  if (!user.avatar) {
    return <AccountCircleIcon sx={{ width, height, ...sx }} />;
  }

  return (
    <Box
      component="img"
      src={user.avatar}
      alt="User Profile"
      sx={{ borderRadius: "50%", width, height, ...sx }}
    />
  );
}
