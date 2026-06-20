type PaginationProps = {
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
};

const Pagination: React.FC<PaginationProps> = ({
  currentPage,
  totalPages,
  onPageChange,
}) => {
  if (totalPages <= 1) {
    return null;
  }

  // Окно из (до) 3 страниц, всегда в пределах [1, totalPages].
  const windowSize = Math.min(3, totalPages);
  let windowStart = Math.max(1, currentPage - 1);
  if (windowStart + windowSize - 1 > totalPages) {
    windowStart = totalPages - windowSize + 1;
  }
  windowStart = Math.max(1, windowStart);
  const pages = Array.from({ length: windowSize }, (_, i) => windowStart + i);

  const hasLeftGap = windowStart > 1;
  const hasRightGap = windowStart + windowSize - 1 < totalPages;

  return (
    <div className="flex items-center ">
      <button
        onClick={() => onPageChange(currentPage - 1)}
        disabled={currentPage <= 1}
        className="mr-2.5 flex items-center h-10 justify-center rounded-lg border border-gray-300 bg-white px-3.5 py-2.5 text-gray-700 shadow-theme-xs hover:bg-gray-50 disabled:opacity-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-white/[0.03] text-sm"
      >
        Previous
      </button>
      <div className="flex items-center gap-2">
        {hasLeftGap && <span className="px-2">...</span>}
        {pages.map((page) => (
          <button
            key={page}
            onClick={() => onPageChange(page)}
            className={`px-4 py-2 rounded ${
              currentPage === page
                ? "bg-brand-500 text-white"
                : "text-gray-700 dark:text-gray-400"
            } flex w-10 items-center justify-center h-10 rounded-lg text-sm font-medium hover:bg-blue-500/[0.08] hover:text-brand-500 dark:hover:text-brand-500`}
          >
            {page}
          </button>
        ))}
        {hasRightGap && <span className="px-2">...</span>}
      </div>
      <button
        onClick={() => onPageChange(currentPage + 1)}
        disabled={currentPage >= totalPages}
        className="ml-2.5 flex items-center justify-center rounded-lg border border-gray-300 bg-white px-3.5 py-2.5 text-gray-700 shadow-theme-xs text-sm hover:bg-gray-50 h-10 disabled:opacity-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-white/[0.03]"
      >
        Next
      </button>
    </div>
  );
};

export default Pagination;
