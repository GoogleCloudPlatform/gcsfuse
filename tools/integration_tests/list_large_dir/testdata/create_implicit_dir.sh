a=0
#Iterate the loop until a greater than 10
touch a.txt
while [ $a -lt 100 ]
do
   dir="Test"$a
   a=`expr $a + 1`
   gsutil cp a.txt gs://$1/$dir/
done
