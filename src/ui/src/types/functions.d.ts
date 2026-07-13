type errorWithJsx = null | string | JsxElement | Array<JsxElement>;

declare type SetErrorWithJSX = (val: errorWithJsx) => void;
declare type SetError = (val: null | string | JsxElement) => void;
declare type SetLoading = (val: boolean) => void;
declare type CloseModal = (cb: () => void) => void;
