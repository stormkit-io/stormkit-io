import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";

interface HelpTableRow {
  name: React.ReactNode;
  desc: React.ReactNode;
}

interface Props {
  items: HelpTableRow[];
}

export default function HelpTable({ items }: Props) {
  return (
    <Box
      component="table"
      sx={{
        width: "100%",
        borderCollapse: "collapse",
        "& th, & td": {
          textAlign: "left",
          p: 1,
          borderBottom: "1px solid",
          borderColor: "container.border",
        },
        "& th": {
          fontWeight: "bold",
        },
        "& th:first-of-type, & td:first-of-type": {
          whiteSpace: "nowrap",
          width: "120px",
          maxWidth: "120px",
        },
        "& tbody tr:last-of-type td": {
          borderBottom: "none",
        },
      }}
    >
      <Box component="thead">
        <Box component="tr">
          <Typography component="th">Name</Typography>
          <Typography component="th">Description</Typography>
        </Box>
      </Box>
      <Box component="tbody">
        {items.map((item, index) => (
          <Box component="tr" key={index}>
            <Typography component="td">{item.name}</Typography>
            <Typography component="td">{item.desc}</Typography>
          </Box>
        ))}
      </Box>
    </Box>
  );
}
