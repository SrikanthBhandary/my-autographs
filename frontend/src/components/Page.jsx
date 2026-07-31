import { forwardRef } from "react";

const Page = forwardRef(({ className = "", children }, ref) => (
  <div className={`book-leaf-page ${className}`} ref={ref}>
    {children}
  </div>
));

export default Page;
