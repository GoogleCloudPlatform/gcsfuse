bash ./write_96thread_256K_1M_120M_500M_1G.sh | tee ~/write_96thread_256K_1M_120M_500M_1G_r1.txt
sleep 300
bash ./write_96thread_256K_1M_120M_500M_1G.sh | tee ~/write_96thread_256K_1M_120M_500M_1G_r2.txt
sleep 300
bash ./write_96thread_256K_1M_120M_500M_1G.sh | tee ~/write_96thread_256K_1M_120M_500M_1G_r3.txt
sleep 300
bash ./write_48thread_256K_1M_120M_500M_1G.sh | tee ~/write_48thread_256K_1M_120M_500M_1G_r1.txt
sleep 300
bash ./write_48thread_256K_1M_120M_500M_1G.sh | tee ~/write_48thread_256K_1M_120M_500M_1G_r2.txt
sleep 300
bash ./write_48thread_256K_1M_120M_500M_1G.sh | tee ~/write_48thread_256K_1M_120M_500M_1G_r3.txt
sleep 300
bash ./write_16thread_256K_1M_120M_500M_1G.sh | tee ~/write_16thread_256K_1M_120M_500M_1G_r1.txt
sleep 300
bash ./write_16thread_256K_1M_120M_500M_1G.sh | tee ~/write_16thread_256K_1M_120M_500M_1G_r2.txt
sleep 300
bash ./write_16thread_256K_1M_120M_500M_1G.sh | tee ~/write_16thread_256K_1M_120M_500M_1G_r3.txt
sleep 300