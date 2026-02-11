export default (): Snippet[] => [
  {
    id: "1",
    enabled: false,
    prepend: false,
    location: "head",
    content: "<script>console.log('snippet 1')</script>",
    title: "Snippet 1",
  },
  {
    id: "2",
    enabled: true,
    prepend: false,
    location: "body",
    content: "<script>console.log('snippet 2')</script>",
    title: "Snippet 2",
  },
  {
    id: "3",
    enabled: true,
    prepend: true,
    location: "head",
    content: "<script>console.log('snippet 3')</script>",
    title: "Snippet 3",
  },
  {
    id: "4",
    enabled: true,
    prepend: true,
    location: "body",
    content: "<script>console.log('snippet 4')</script>",
    title: "Snippet 4",
  },
];
