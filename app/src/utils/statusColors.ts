export function taskStatusColor(tag: string): string {
  switch (tag) {
    case "Pending":
      return "bg-gray-800 text-gray-300";
    case "Assigned":
      return "bg-blue-900/50 text-blue-300";
    case "Running":
      return "bg-yellow-900/50 text-yellow-300";
    case "Completed":
      return "bg-green-900/50 text-green-300";
    case "Failed":
      return "bg-red-900/50 text-red-300";
    case "Cancelled":
      return "bg-gray-800 text-gray-500";
    default:
      return "bg-gray-800 text-gray-400";
  }
}
